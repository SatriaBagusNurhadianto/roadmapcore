package progress

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func newRepo(t *testing.T) *Repository {
	t.Helper()
	dir := t.TempDir()
	r, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return r
}

func initPath(t *testing.T, r *Repository) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.UpdateProgress(ctx, func(f *File) error {
		f.Tracker.ActivePath = "data-analyst"
		f.Tracker.Path("data-analyst")
		return nil
	}); err != nil {
		t.Fatalf("init: %v", err)
	}
}

func TestUpdateAndLoad(t *testing.T) {
	r := newRepo(t)
	initPath(t, r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.UpdateProgress(ctx, func(f *File) error {
		p := f.Tracker.Path("data-analyst")
		return p.SetWeekStatus(2, WeekDone, f.UpdatedAt)
	}); err != nil {
		t.Fatalf("UpdateProgress: %v", err)
	}

	got, err := r.LoadProgress(ctx)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	p := got.Tracker.Paths["data-analyst"]
	if p.WeekStatusOf(2) != WeekDone {
		t.Fatalf("status minggu 2 = %q", p.WeekStatusOf(2))
	}
	if got.SchemaVersion != SchemaVersion {
		t.Fatalf("schema = %d", got.SchemaVersion)
	}
	if got.UpdatedAt.IsZero() {
		t.Fatal("updated_at tidak diisi")
	}
}

func TestOwnerSeparation(t *testing.T) {
	r := newRepo(t)
	initPath(t, r)
	ctx := context.Background()

	// Tulis section reminder dulu.
	if err := r.UpdateProgress(ctx, func(f *File) error {
		f.Reminder.Streak = Streak{Count: 4, LastActiveDate: "2026-11-16"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	before, err := r.LoadProgress(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Trace Learn mengubah tracker saja.
	if err := r.UpdateProgress(ctx, func(f *File) error {
		p := f.Tracker.Path("data-analyst")
		return p.SetWeekStatus(1, WeekDone, f.UpdatedAt)
	}); err != nil {
		t.Fatal(err)
	}
	after, err := r.LoadProgress(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if after.Reminder.Streak != before.Reminder.Streak {
		t.Fatalf("section reminder berubah: %+v -> %+v", before.Reminder.Streak, after.Reminder.Streak)
	}
	if after.Reminder.Streak.Count != 4 {
		t.Fatalf("streak = %+v", after.Reminder.Streak)
	}
}

func TestUpdateSettings(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	if err := r.UpdateSettings(ctx, func(s *Settings) error {
		s.Reminder["jam"] = "19:00"
		return nil
	}); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	s, err := r.LoadSettings(ctx)
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if s.Reminder["jam"] != "19:00" {
		t.Fatalf("reminder[jam] = %v", s.Reminder["jam"])
	}
}

func TestConcurrentUpdatesNoLostUpdate(t *testing.T) {
	r := newRepo(t)
	initPath(t, r)
	ctx := context.Background()

	const workers = 8
	const each = 10
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			// Deadline longgar per panggilan: kontensi 8 penulis bersifat
			// sintetis, dan yang diuji di sini adalah tidak adanya update hilang.
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			for i := 0; i < each; i++ {
				id := fmt.Sprintf("w%d-%d", w, i)
				if err := r.UpdateProgress(ctx, func(f *File) error {
					f.Reminder.CustomTasks = append(f.Reminder.CustomTasks, CustomTask{
						ID: id, Title: "tugas", Date: "2026-11-17",
					})
					return nil
				}); err != nil {
					errs <- err
					return
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("update bersamaan: %v", err)
	}

	got, err := r.LoadProgress(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Reminder.CustomTasks) != workers*each {
		t.Fatalf("custom task = %d, want %d (ada yang hilang)", len(got.Reminder.CustomTasks), workers*each)
	}
	seen := map[string]bool{}
	for _, ct := range got.Reminder.CustomTasks {
		if seen[ct.ID] {
			t.Fatalf("ID duplikat: %s", ct.ID)
		}
		seen[ct.ID] = true
	}
}

func TestCorruptFilePreserved(t *testing.T) {
	r := newRepo(t)
	initPath(t, r)
	ctx := context.Background()

	before, err := os.ReadFile(r.progressPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(r.progressPath(), []byte("{ ini bukan json"), 0o600); err != nil {
		t.Fatal(err)
	}

	err = r.UpdateProgress(ctx, func(f *File) error { return nil })
	var corrupt *ErrCorrupt
	if !errors.As(err, &corrupt) {
		t.Fatalf("UpdateProgress = %v, want ErrCorrupt", err)
	}
	after, err := os.ReadFile(r.progressPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != "{ ini bukan json" {
		t.Fatal("file rusak boleh diubah/dihapus")
	}
	if len(before) == 0 {
		t.Fatal("ok")
	}
	// tidak ada sisa file temp
	matches, _ := filepath.Glob(filepath.Join(r.Dir(), "*.tmp-*"))
	if len(matches) != 0 {
		t.Fatalf("sisa file temp: %v", matches)
	}
}

func TestBackupRotated(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	initPath(t, r)

	first, err := os.ReadFile(r.progressPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := r.UpdateProgress(ctx, func(f *File) error {
		f.Tracker.ActivePath = "swe-js"
		f.Tracker.Path("swe-js")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	bak, err := os.ReadFile(r.progressPath() + ".bak")
	if err != nil {
		t.Fatalf("backup tidak ada: %v", err)
	}
	if string(bak) != string(first) {
		t.Fatal("isi backup bukan versi sebelumnya")
	}
}

func TestNoTempFilesLeft(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	initPath(t, r)
	for i := 0; i < 5; i++ {
		if err := r.UpdateProgress(ctx, func(f *File) error {
			p := f.Tracker.Path("data-analyst")
			return p.SetWeekStatus(i+1, WeekInProgress, f.UpdatedAt)
		}); err != nil {
			t.Fatal(err)
		}
	}
	matches, err := filepath.Glob(filepath.Join(r.Dir(), "*.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("sisa file temp: %v", matches)
	}
}

func TestSchemaTooNew(t *testing.T) {
	r := newRepo(t)
	initPath(t, r)
	path := r.progressPath()
	if err := os.WriteFile(path, []byte(`{"schema_version":99,"tracker":{"active_path":"x","paths":{"x":{}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := r.LoadProgress(context.Background()); !errors.Is(err, ErrSchemaTooNew) {
		t.Fatalf("LoadProgress = %v, want ErrSchemaTooNew", err)
	}
	ctx := context.Background()
	if err := r.UpdateProgress(ctx, func(f *File) error { return nil }); !errors.Is(err, ErrSchemaTooNew) {
		t.Fatalf("UpdateProgress = %v, want ErrSchemaTooNew", err)
	}
}

func TestLoadNotExist(t *testing.T) {
	r := newRepo(t)
	if _, err := r.LoadProgress(context.Background()); !errors.Is(err, ErrNotExist) {
		t.Fatalf("LoadProgress = %v, want ErrNotExist", err)
	}
}

func TestRecoverFromBackup(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	initPath(t, r)
	// .bak berisi versi sebelumnya, jadi lakukan satu update lagi setelah
	// status minggu diisi agar versi baik tersimpan sebagai backup.
	if err := r.UpdateProgress(ctx, func(f *File) error {
		f.Tracker.Path("data-analyst").SetWeekStatus(1, WeekDone, f.UpdatedAt)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := r.UpdateProgress(ctx, func(f *File) error {
		f.Reminder.Streak = Streak{Count: 2, LastActiveDate: "2026-11-17"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(r.progressPath(), []byte("rusak"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := r.RecoverFromBackup(ctx); err != nil {
		t.Fatalf("RecoverFromBackup: %v", err)
	}
	got, err := r.LoadProgress(ctx)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if got.Tracker.Paths["data-analyst"].WeekStatusOf(1) != WeekDone {
		t.Fatal("backup tidak memulihkan status minggu")
	}
}

func TestMigrateFromZero(t *testing.T) {
	f := &File{}
	if err := Migrate(f); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if f.SchemaVersion != SchemaVersion {
		t.Fatalf("schema = %d", f.SchemaVersion)
	}
}

func TestAtomicWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "progress.json")
	if err := writeFileAtomic(path, []byte("v1\n")); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(path, []byte("v2\n")); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "v2\n" {
		t.Fatalf("isi = %q", got)
	}
	bak, _ := os.ReadFile(path + ".bak")
	if string(bak) != "v1\n" {
		t.Fatalf("backup = %q", bak)
	}
	// permission 0600 — hanya relevan di Unix; Windows tidak punya bit POSIX.
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Fatalf("permission = %o, want 600", perm)
		}
	}
}

// --- bukti dua proses ---

func TestHelperProcess(t *testing.T) {
	mode := os.Getenv("ROADMAPCORE_HELPER")
	if mode == "" {
		return
	}
	dir := os.Getenv("ROADMAPCORE_DIR")
	r, err := Open(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	switch mode {
	case "hold":
		unlock, err := acquire(context.Background(), filepath.Join(dir, "progress.lock"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(3)
		}
		defer unlock()
		// sinyal siap dikirim setelah lock benar-benar dipegang
		fmt.Println("ready")
		os.Stdout.Sync()
		time.Sleep(15 * time.Second)
	case "writer":
		fmt.Println("ready")
		os.Stdout.Sync()
		n := 20
		for i := 0; i < n; i++ {
			id := os.Getenv("ROADMAPCORE_ID")
			if err := r.UpdateProgress(context.Background(), func(f *File) error {
				f.Reminder.CustomTasks = append(f.Reminder.CustomTasks, CustomTask{
					ID: fmt.Sprintf("%s-%d", id, i), Title: "tugas", Date: "2026-11-17",
				})
				return nil
			}); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(4)
			}
		}
		fmt.Println("done")
		os.Stdout.Sync()
	}
	os.Exit(0)
}

func startHelper(t *testing.T, mode, dir, id string) (*exec.Cmd, *bufio.Reader) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess")
	cmd.Env = append(os.Environ(),
		"ROADMAPCORE_HELPER="+mode,
		"ROADMAPCORE_DIR="+dir,
		"ROADMAPCORE_ID="+id,
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	br := bufio.NewReader(stdout)
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		line, err := br.ReadString('\n')
		if strings.Contains(line, "ready") {
			return cmd, br
		}
		if err != nil {
			t.Fatalf("helper berhenti lebih awal: %v", err)
		}
	}
	t.Fatal("helper tidak siap")
	return nil, nil
}

func waitDone(t *testing.T, cmd *exec.Cmd, br *bufio.Reader) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		line, err := br.ReadString('\n')
		if strings.Contains(line, "done") {
			select {
			case <-done:
			case <-time.After(5 * time.Second):
			}
			return
		}
		if err != nil {
			t.Fatalf("helper berhenti lebih awal: %v", err)
		}
	}
	t.Fatal("helper tidak selesai")
}

func TestTwoProcessNoLostUpdate(t *testing.T) {
	r := newRepo(t)
	initPath(t, r)
	dir := r.Dir()

	const each = 20
	cmd1, br1 := startHelper(t, "writer", dir, "p1")
	cmd2, br2 := startHelper(t, "writer", dir, "p2")
	waitDone(t, cmd1, br1)
	waitDone(t, cmd2, br2)

	got, err := r.LoadProgress(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Reminder.CustomTasks) != 2*each {
		t.Fatalf("custom task = %d, want %d (ada yang hilang)", len(got.Reminder.CustomTasks), 2*each)
	}
	seen := map[string]bool{}
	for _, ct := range got.Reminder.CustomTasks {
		if seen[ct.ID] {
			t.Fatalf("ID duplikat: %s", ct.ID)
		}
		seen[ct.ID] = true
	}
}

func TestLockTimeoutActionable(t *testing.T) {
	r := newRepo(t)
	initPath(t, r)

	cmd, br := startHelper(t, "hold", r.Dir(), "")
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = br.ReadString('\n')
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	err := r.UpdateProgress(ctx, func(f *File) error { return nil })
	if err == nil {
		t.Fatal("UpdateProgress seharusnya gagal karena lock dipegang proses lain")
	}
	if !strings.Contains(err.Error(), "sedang dipakai") {
		t.Fatalf("pesan error tidak actionable: %v", err)
	}
}

func TestLockFileTetapAda(t *testing.T) {
	r := newRepo(t)
	initPath(t, r)
	if _, err := os.Stat(filepath.Join(r.Dir(), "progress.lock")); err != nil {
		t.Fatalf("lock file harus tetap ada: %v", err)
	}
	entries, err := os.ReadDir(r.Dir())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	joined := strings.Join(names, ",")
	for _, want := range []string{"progress.json", "progress.lock"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("file %s tidak ada; isi: %s", want, joined)
		}
	}
}
