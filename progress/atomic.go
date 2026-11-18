package progress

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// writeTempFileAtomic menuliskan data ke path secara atomik tanpa memutar backup.
// Langkah: temp di direktori yang sama -> sync -> close -> rename.
func writeTempFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	tmp, err := os.CreateTemp(dir, base+".tmp-*")
	if err != nil {
		return fmt.Errorf("buat temp %s: %w", path, err)
	}
	tmpName := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpName)
		}
	}()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod %s: %w", tmpName, err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("tulis %s: %w", tmpName, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("tutup %s: %w", tmpName, err)
	}
	// os.Rename mengganti target yang sudah ada (terukur di Windows 11 + Go 1.27).
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", tmpName, path, err)
	}
	committed = true
	syncDir(dir)
	return nil
}

// writeFileAtomic menulis data ke path dan memutar salinan lama ke path.bak.
func writeFileAtomic(path string, data []byte) error {
	if err := rotateBackup(path); err != nil {
		return err
	}
	return writeTempFileAtomic(path, data)
}

// rotateBackup menyalin berkas lama ke <path>.bak. Tidak ada bila berkas belum ada.
func rotateBackup(path string) error {
	old, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("baca backup %s: %w", path, err)
	}
	return writeTempFileAtomic(path+".bak", old)
}

// syncDir memaksa penulisan direktori; best-effort, error diabaikan karena
// tidak semua OS/filesystem mendukung fsync pada direktori.
func syncDir(dir string) {
	d, err := os.Open(dir)
	if err != nil {
		return
	}
	defer d.Close()
	_ = d.Sync()
}

// copyFile menyalin berkas secara sederhana (dipakai untuk pemulihan manual).
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	data, err := io.ReadAll(in)
	if err != nil {
		return err
	}
	return writeTempFileAtomic(dst, data)
}

// backupPath mengembalikan path backup dari sebuah berkas data.
func backupPath(path string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + ".bak"
}
