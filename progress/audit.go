package progress

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// genesisHash dipakai sebagai prev pada event pertama.
const genesisHash = ""

// Event adalah satu baris audit.jsonl.
type Event struct {
	Seq     int64     `json:"seq"`
	ID      string    `json:"id"`
	TS      time.Time `json:"ts"`
	App     string    `json:"app"`
	Event   string    `json:"event"`
	Summary string    `json:"summary"`
	Prev    string    `json:"prev"`
	Hash    string    `json:"hash"`
}

// payload mengembalikan representasi kanonik untuk di-hash (tanpa field hash).
func (e Event) payload() []byte {
	type payload struct {
		Seq     int64     `json:"seq"`
		ID      string    `json:"id"`
		TS      time.Time `json:"ts"`
		App     string    `json:"app"`
		Event   string    `json:"event"`
		Summary string    `json:"summary"`
		Prev    string    `json:"prev"`
	}
	b, _ := json.Marshal(payload{
		Seq:     e.Seq,
		ID:      e.ID,
		TS:      e.TS,
		App:     e.App,
		Event:   e.Event,
		Summary: e.Summary,
		Prev:    e.Prev,
	})
	return b
}

// computeHash menghitung hash rantai untuk event.
func (e Event) computeHash() string {
	sum := sha256.Sum256(append([]byte(e.Prev), e.payload()...))
	return hex.EncodeToString(sum[:])
}

// Validate memeriksa keseluruhan rantai.
// Mengembalikan seq pertama yang tidak cocok (0 bila seluruh rantai valid).
func Validate(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer f.Close()

	prev := genesisHash
	var seq int64
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		seq++
		var e Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return seq, fmt.Errorf("audit baris %d tidak dapat diurai: %w", seq, err)
		}
		if e.Prev != prev {
			return seq, fmt.Errorf("audit baris %d: prev tidak cocok", seq)
		}
		want := e.computeHash()
		if e.Hash != want {
			return seq, fmt.Errorf("audit baris %d: hash tidak cocok", seq)
		}
		prev = e.Hash
	}
	if err := sc.Err(); err != nil {
		return seq, err
	}
	return 0, nil
}

// Append menambahkan event ke audit.jsonl di bawah lock audit.
// Seq, prev, dan hash dihitung di sini; TS dan ID diisi default bila kosong.
func (r *Repository) AppendAudit(ctx context.Context, ev Event) (Event, error) {
	unlock, err := acquire(ctx, r.auditLock())
	if err != nil {
		return Event{}, err
	}
	defer unlock()

	lastSeq, lastHash, err := tailAudit(r.auditPath())
	if err != nil {
		return Event{}, err
	}

	if ev.ID == "" {
		ev.ID = newEventID()
	}
	if ev.TS.IsZero() {
		ev.TS = r.Now()
	}
	ev.Seq = lastSeq + 1
	ev.Prev = lastHash
	ev.Hash = ev.computeHash()

	line, err := json.Marshal(ev)
	if err != nil {
		return Event{}, err
	}
	line = append(line, '\n')

	f, err := os.OpenFile(r.auditPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return Event{}, fmt.Errorf("buka audit: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(line); err != nil {
		return Event{}, fmt.Errorf("tulis audit: %w", err)
	}
	return ev, nil
}

// ValidateAudit memvalidasi rantai hash audit.
func (r *Repository) ValidateAudit(ctx context.Context) (int64, error) {
	return Validate(r.auditPath())
}

// tailAudit mengembalikan seq dan hash event terakhir.
func tailAudit(path string) (int64, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, genesisHash, nil
		}
		return 0, "", err
	}
	var last Event
	var seq int64
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		seq++
		if err := json.Unmarshal([]byte(line), &last); err != nil {
			return 0, "", fmt.Errorf("audit baris %d tidak dapat diurai: %w", seq, err)
		}
	}
	return seq, last.Hash, nil
}

func newEventID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
