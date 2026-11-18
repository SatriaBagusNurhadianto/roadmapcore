package progress

import (
	"strings"
	"testing"
	"time"
)

func TestParseWeekStatus(t *testing.T) {
	for _, s := range []WeekStatus{WeekNotStarted, WeekInProgress, WeekWaiting, WeekDone} {
		got, err := ParseWeekStatus(string(s))
		if err != nil || got != s {
			t.Fatalf("ParseWeekStatus(%q) = %q, %v", s, got, err)
		}
	}
	if _, err := ParseWeekStatus("selesai"); err == nil {
		t.Fatal("status ilegal seharusnya ditolak")
	}
}

func TestMiniProjectTransitions(t *testing.T) {
	allowed := map[MiniProjectStatus][]MiniProjectStatus{
		MPEmpty:   {MPWaiting},
		MPWaiting: {MPDone, MPEmpty},
		MPDone:    {MPWaiting, MPEmpty},
	}
	all := []MiniProjectStatus{MPEmpty, MPWaiting, MPDone}
	for from, ok := range allowed {
		for _, to := range all {
			m := MiniProject{Status: from}
			err := m.Transition(to, time.Now())
			shouldOK := false
			for _, a := range ok {
				if a == to {
					shouldOK = true
				}
			}
			if shouldOK && err != nil {
				t.Fatalf("%s -> %s seharusnya diizinkan: %v", from, to, err)
			}
			if !shouldOK && err == nil {
				t.Fatalf("%s -> %s seharusnya ditolak", from, to)
			}
		}
	}
}

func TestActiveWeekAndDoneWeeks(t *testing.T) {
	p := &PathProgress{Weeks: map[string]*WeekState{}}
	if got := p.ActiveWeekUpTo(12); got != 1 {
		t.Fatalf("ActiveWeek = %d, want 1", got)
	}
	now := time.Now().UTC()
	for i := 1; i <= 3; i++ {
		if err := p.SetWeekStatus(i, WeekDone, now); err != nil {
			t.Fatal(err)
		}
	}
	if got := p.ActiveWeekUpTo(12); got != 4 {
		t.Fatalf("ActiveWeek = %d, want 4", got)
	}
	if got := p.DoneWeeksUpTo(12); got != 3 {
		t.Fatalf("DoneWeeks = %d, want 3", got)
	}
	// done -> in_progress: minggu aktif kembali 3
	if err := p.SetWeekStatus(3, WeekInProgress, now); err != nil {
		t.Fatal(err)
	}
	if got := p.ActiveWeekUpTo(12); got != 3 {
		t.Fatalf("ActiveWeek = %d, want 3", got)
	}
}

func TestNormalizeDefaults(t *testing.T) {
	f := NewFile()
	if f.Reminder.Focus.DurationMin != 25 {
		t.Fatalf("focus default = %d", f.Reminder.Focus.DurationMin)
	}
	f.Tracker.ActivePath = "data-analyst"
	f.Tracker.Path("data-analyst")
	if err := f.Normalize(); err != nil {
		t.Fatal(err)
	}
	if f.Tracker.Paths["data-analyst"].MiniProject.Status != MPEmpty {
		t.Fatalf("mini project default = %q", f.Tracker.Paths["data-analyst"].MiniProject.Status)
	}
}

func TestNormalizeRejectsBadWeekKey(t *testing.T) {
	f := NewFile()
	f.Tracker.ActivePath = "x"
	p := f.Tracker.Path("x")
	p.Weeks["minggu-satu"] = &WeekState{}
	if err := f.Normalize(); err == nil {
		t.Fatal("kunci minggu non-angka seharusnya ditolak")
	}
}

func TestValidateRejectsUnknownStatus(t *testing.T) {
	f := NewFile()
	f.Tracker.ActivePath = "x"
	p := f.Tracker.Path("x")
	p.Weeks["1"] = &WeekState{Status: WeekStatus("nope")}
	if err := f.Normalize(); err != nil {
		t.Fatal(err)
	}
	if err := f.Validate(); err == nil || !strings.Contains(err.Error(), "tidak valid") {
		t.Fatalf("Validate = %v, want penolakan status", err)
	}
}

func TestValidateRejectsMissingActivePath(t *testing.T) {
	f := NewFile()
	if err := f.Normalize(); err != nil {
		t.Fatal(err)
	}
	if err := f.Validate(); err == nil {
		t.Fatal("active_path kosong seharusnya ditolak")
	}
}

func TestValidateLevel(t *testing.T) {
	for _, l := range []string{"pemula", "menengah_awal", "di_atas_rata_rata"} {
		if err := ValidateLevel(l); err != nil {
			t.Fatalf("ValidateLevel(%q): %v", l, err)
		}
	}
	if err := ValidateLevel("master"); err == nil {
		t.Fatal("level tidak dikenal seharusnya ditolak")
	}
}

func TestNormalizePaksaUTC(t *testing.T) {
	wib := time.FixedZone("WIB", 7*3600)
	f := NewFile()
	f.Tracker.ActivePath = "x"
	p := f.Tracker.Path("x")
	if err := p.SetWeekStatus(1, WeekDone, time.Date(2026, 11, 17, 23, 0, 0, 0, wib)); err != nil {
		t.Fatal(err)
	}
	if err := f.Normalize(); err != nil {
		t.Fatal(err)
	}
	got := p.Weeks["1"].UpdatedAt
	if got.Location() != time.UTC {
		t.Fatalf("lokasi timestamp = %v, want UTC", got.Location())
	}
	if got.Hour() != 16 {
		t.Fatalf("jam UTC = %d, want 16", got.Hour())
	}
	if f.Reminder.CustomTasks == nil {
		t.Fatal("custom tasks harus slice kosong, bukan null")
	}
}

func TestWeekStatusZeroValueNormalizes(t *testing.T) {
	f := NewFile()
	f.Tracker.ActivePath = "x"
	p := f.Tracker.Path("x")
	p.Weeks["1"] = &WeekState{}
	if err := f.Normalize(); err != nil {
		t.Fatal(err)
	}
	if got := p.WeekStatusOf(1); got != WeekNotStarted {
		t.Fatalf("status = %q, want not_started", got)
	}
}
