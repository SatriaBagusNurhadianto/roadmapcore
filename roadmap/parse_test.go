package roadmap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseValid(t *testing.T) {
	r, err := ParseFile("testdata/valid.yaml")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if r.ID != "contoh-jalur" {
		t.Fatalf("ID = %q", r.ID)
	}
	if got := len(r.WeekIndices()); got != ExpectedWeeks {
		t.Fatalf("jumlah minggu = %d, want %d", got, ExpectedWeeks)
	}
	if w, ok := r.FindWeek(7); !ok || w.Title != "Minggu 7" {
		t.Fatalf("FindWeek(7) = %+v, %v", w, ok)
	}
	if _, ok := r.FindWeek(13); ok {
		t.Fatal("FindWeek(13) seharusnya tidak ada")
	}
}

func TestLoadDir(t *testing.T) {
	list, err := LoadDir("testdata/load")
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name string
		file string
	}{
		{"id ilegal", "invalid_id.yaml"},
		{"minggu 11 saja", "invalid_weeks.yaml"},
		{"minggu duplikat", "invalid_dup.yaml"},
		{"estimasi nol", "invalid_estimate.yaml"},
		{"indikator salah", "invalid_indicator.yaml"},
		{"field tak dikenal", "invalid_field.yaml"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := ParseFile(filepath.Join("testdata", c.file)); err == nil {
				t.Fatalf("ParseFile(%s) seharusnya gagal", c.file)
			}
		})
	}
}

func TestParseEmpty(t *testing.T) {
	if _, err := Parse(nil); err == nil {
		t.Fatal("Parse(nil) seharusnya gagal")
	}
}

func TestLoadBuiltinTanpaRoadmap(t *testing.T) {
	// Direktori embed belum berisi YAML roadmap asli. Keadaan kosong bukan
	// error: aplikasi tetap bisa memakai roadmap dari direktori data.
	list, err := LoadBuiltin()
	if err != nil {
		t.Fatalf("LoadBuiltin: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("len = %d, want 0", len(list))
	}
}

func TestLoadEmbeddedDariFSUji(t *testing.T) {
	fsys := os.DirFS("testdata/load")
	list, err := LoadEmbedded(fsys)
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
}
