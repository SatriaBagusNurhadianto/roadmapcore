package progress

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"
)

// ErrNotExist dikembalikan bila berkas data belum ada.
var ErrNotExist = errors.New("berkas data belum ada")

// ErrCorrupt menandai berkas data tidak dapat diurai. Berkas asli tidak pernah dihapus.
type ErrCorrupt struct {
	Path string
	Err  error
}

func (e *ErrCorrupt) Error() string {
	return fmt.Sprintf("berkas rusak %s: %v", e.Path, e.Err)
}

func (e *ErrCorrupt) Unwrap() error { return e.Err }

// ErrSchemaTooNew menandai schema lebih baru dari versi aplikasi.
var ErrSchemaTooNew = errors.New("schema lebih baru dari versi aplikasi")

// Repository mengelola direktori data bersama.
type Repository struct {
	dir string
	now func() time.Time
}

// Open membuka (dan membuat bila perlu) direktori data.
func Open(dir string) (*Repository, error) {
	if dir == "" {
		return nil, fmt.Errorf("direktori data wajib diisi")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("buat direktori %s: %w", dir, err)
	}
	return &Repository{dir: dir, now: time.Now}, nil
}

// Dir mengembalikan direktori data.
func (r *Repository) Dir() string { return r.dir }

// Now mengembalikan waktu sekarang dalam UTC.
func (r *Repository) Now() time.Time { return r.now().UTC() }

func (r *Repository) progressPath() string { return r.dir + string(os.PathSeparator) + "progress.json" }
func (r *Repository) settingsPath() string { return r.dir + string(os.PathSeparator) + "settings.json" }
func (r *Repository) progressLock() string { return r.dir + string(os.PathSeparator) + "progress.lock" }
func (r *Repository) settingsLock() string { return r.dir + string(os.PathSeparator) + "settings.lock" }
func (r *Repository) auditPath() string    { return r.dir + string(os.PathSeparator) + "audit.jsonl" }
func (r *Repository) auditLock() string    { return r.dir + string(os.PathSeparator) + "audit.lock" }

// LockFile memperoleh lock pada berkas kunci sembarang dan mengembalikan
// fungsi lepas. Dipakai daemon untuk lock single-instance (`daemon.lock`).
//
// Beda dengan lock internal: LockFile menolak cepat bila lock sudah dipegang
// proses yang sama. Pemeriksaan single-instance harus gagal segera, bukan
// menunggu pemilik melepas lock.
func LockFile(ctx context.Context, path string) (func(), error) {
	m := lockMutex(path)
	if !m.TryLock() {
		return nil, fmt.Errorf("lock %s sudah dipegang proses ini", path)
	}
	return acquireHeld(ctx, path, m)
}

// AuditPath dipakai adapter untuk menampilkan lokasi log.
func (r *Repository) AuditPath() string { return r.auditPath() }

// UpdateProgress mengubah section tracker di bawah lock.
func (r *Repository) UpdateProgress(ctx context.Context, fn func(*File) error) error {
	var dst File
	return r.update(ctx, r.progressPath(), r.progressLock(), &dst, func() error {
		if dst.SchemaVersion == 0 {
			dst = *NewFile()
		}
		if err := Migrate(&dst); err != nil {
			return err
		}
		if err := fn(&dst); err != nil {
			return err
		}
		if err := dst.Normalize(); err != nil {
			return err
		}
		dst.SchemaVersion = SchemaVersion
		dst.UpdatedAt = r.Now()
		return dst.Validate()
	})
}

// UpdateSettings mengubah section reminder/tracker settings di bawah lock.
func (r *Repository) UpdateSettings(ctx context.Context, fn func(*Settings) error) error {
	var dst Settings
	return r.update(ctx, r.settingsPath(), r.settingsLock(), &dst, func() error {
		if dst.SchemaVersion == 0 {
			dst = *NewSettings()
		}
		if err := MigrateSettings(&dst); err != nil {
			return err
		}
		if err := fn(&dst); err != nil {
			return err
		}
		dst.Normalize()
		dst.SchemaVersion = SchemaVersion
		dst.UpdatedAt = r.Now()
		return dst.Validate()
	})
}

// update adalah inti read-modify-write: lock -> baca -> fn -> tulis atomik.
// fn dipanggil dengan dst yang sudah berisi data lama (atau default bila baru).
func (r *Repository) update(ctx context.Context, file, lockFile string, dst any, fn func() error) error {
	unlock, err := acquire(ctx, lockFile)
	if err != nil {
		return err
	}
	defer unlock()

	data, err := os.ReadFile(file)
	switch {
	case err == nil:
		if err := json.Unmarshal(data, dst); err != nil {
			return &ErrCorrupt{Path: file, Err: err}
		}
	case errors.Is(err, fs.ErrNotExist):
		// berkas belum ada: pakai nilai default
	default:
		return fmt.Errorf("baca %s: %w", file, err)
	}

	if err := fn(); err != nil {
		return err
	}
	out, err := json.MarshalIndent(dst, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", file, err)
	}
	out = append(out, '\n')
	if err := writeFileAtomic(file, out); err != nil {
		return err
	}
	return nil
}

// LoadProgress membaca progress.json tanpa lock (rename atomik membuat pembacaan aman).
func (r *Repository) LoadProgress(ctx context.Context) (*File, error) {
	data, err := os.ReadFile(r.progressPath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrNotExist
		}
		return nil, fmt.Errorf("baca %s: %w", r.progressPath(), err)
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, &ErrCorrupt{Path: r.progressPath(), Err: err}
	}
	if f.SchemaVersion > SchemaVersion {
		return nil, fmt.Errorf("%w: %d > %d", ErrSchemaTooNew, f.SchemaVersion, SchemaVersion)
	}
	if f.SchemaVersion == 0 {
		f = *NewFile()
	}
	if err := f.Normalize(); err != nil {
		return nil, err
	}
	return &f, nil
}

// LoadSettings membaca settings.json.
func (r *Repository) LoadSettings(ctx context.Context) (*Settings, error) {
	data, err := os.ReadFile(r.settingsPath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrNotExist
		}
		return nil, fmt.Errorf("baca %s: %w", r.settingsPath(), err)
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, &ErrCorrupt{Path: r.settingsPath(), Err: err}
	}
	if s.SchemaVersion > SchemaVersion {
		return nil, fmt.Errorf("%w: %d > %d", ErrSchemaTooNew, s.SchemaVersion, SchemaVersion)
	}
	if s.SchemaVersion == 0 {
		s = *NewSettings()
	}
	s.Normalize()
	return &s, nil
}

// RecoverFromBackup mencoba memulihkan progress.json dari .bak.
// Berkas rusak tidak pernah dihapus; pemanggil yang memutuskan.
func (r *Repository) RecoverFromBackup(ctx context.Context) error {
	unlock, err := acquire(ctx, r.progressLock())
	if err != nil {
		return err
	}
	defer unlock()

	bak := r.progressPath() + ".bak"
	data, err := os.ReadFile(bak)
	if err != nil {
		return fmt.Errorf("baca backup %s: %w", bak, err)
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("backup %s juga rusak: %w", bak, err)
	}
	return writeFileAtomic(r.progressPath(), data)
}
