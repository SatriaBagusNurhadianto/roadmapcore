package progress

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppendAndValidate(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		ev, err := r.AppendAudit(ctx, Event{
			App:     "tracker",
			Event:   "week.status",
			Summary: "minggu 1 -> done",
		})
		if err != nil {
			t.Fatalf("AppendAudit: %v", err)
		}
		if ev.Seq != int64(i+1) {
			t.Fatalf("seq = %d, want %d", ev.Seq, i+1)
		}
		if ev.Hash == "" || ev.ID == "" || ev.TS.IsZero() {
			t.Fatalf("field tidak diisi: %+v", ev)
		}
		if i == 0 && ev.Prev != "" {
			t.Fatalf("prev event pertama = %q, want kosong", ev.Prev)
		}
	}

	bad, err := r.ValidateAudit(ctx)
	if err != nil {
		t.Fatalf("ValidateAudit: %v", err)
	}
	if bad != 0 {
		t.Fatalf("firstBad = %d, want 0", bad)
	}
}

func TestAuditChainTerhubung(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	e1, err := r.AppendAudit(ctx, Event{App: "tracker", Event: "a", Summary: "1"})
	if err != nil {
		t.Fatal(err)
	}
	e2, err := r.AppendAudit(ctx, Event{App: "tracker", Event: "b", Summary: "2"})
	if err != nil {
		t.Fatal(err)
	}
	if e2.Prev != e1.Hash {
		t.Fatalf("prev event 2 = %q, want %q", e2.Prev, e1.Hash)
	}
}

func TestAuditTamperDetected(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, err := r.AppendAudit(ctx, Event{App: "tracker", Event: "a", Summary: "x"}); err != nil {
			t.Fatal(err)
		}
	}
	path := r.auditPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	// ubah summary baris ke-2 tanpa menghitung ulang hash
	lines[1] = strings.Replace(lines[1], `"x"`, `"y"`, 1)
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	bad, err := r.ValidateAudit(ctx)
	if bad != 2 {
		t.Fatalf("firstBad = %d, want 2 (err=%v)", bad, err)
	}
}

func TestAuditKosongValid(t *testing.T) {
	r := newRepo(t)
	bad, err := r.ValidateAudit(context.Background())
	if err != nil || bad != 0 {
		t.Fatalf("ValidateAudit kosong = %d, %v", bad, err)
	}
}

func TestAuditRusakDitunukkanSeq(t *testing.T) {
	r := newRepo(t)
	if err := os.WriteFile(r.auditPath(), []byte("bukan json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	bad, err := r.ValidateAudit(context.Background())
	if bad != 1 {
		t.Fatalf("firstBad = %d, want 1 (err=%v)", bad, err)
	}
}

func TestAuditHashStabil(t *testing.T) {
	e := Event{Seq: 1, ID: "id", TS: time.Unix(0, 0).UTC(), App: "a", Event: "b", Summary: "c", Prev: "p"}
	if e.computeHash() != e.computeHash() {
		t.Fatal("hash tidak stabil")
	}
	changed := e
	changed.Summary = "d"
	if e.computeHash() == changed.computeHash() {
		t.Fatal("hash harus berubah bila payload berubah")
	}
	other := e
	other.Prev = "q"
	if e.computeHash() == other.computeHash() {
		t.Fatal("hash harus berubah bila prev berubah")
	}
}

func TestAuditFileAdaDiDir(t *testing.T) {
	r := newRepo(t)
	if _, err := os.Stat(filepath.Join(r.Dir(), "audit.jsonl")); err == nil {
		t.Fatal("audit.jsonl seharusnya belum ada sebelum append")
	}
	if _, err := r.AppendAudit(context.Background(), Event{App: "a", Event: "b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(r.Dir(), "audit.jsonl")); err != nil {
		t.Fatalf("audit.jsonl tidak ada: %v", err)
	}
}
