package roadmap

import (
	"os"
	"path/filepath"
	"strconv"
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
	if got := len(r.WeekIndices()); got != r.TotalWeeks() {
		t.Fatalf("jumlah minggu = %d, want %d", got, r.TotalWeeks())
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
		{"indeks minggu tidak kontigu", "invalid_weeks.yaml"},
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

func TestLoadBuiltinDuaRoadmap(t *testing.T) {
	// Roadmap bawaan diambil dari dokumen sumber. Bila daftar ini berubah,
	// ubah juga README dan docs/FORMAT.md.
	list, err := LoadBuiltin()
	if err != nil {
		t.Fatalf("LoadBuiltin: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2", len(list))
	}
	ida := map[string]bool{}
	for _, r := range list {
		ida[r.ID] = true
	}
	for _, want := range []string{"data-analyst", "swe-javascript"} {
		if !ida[want] {
			t.Fatalf("roadmap %q tidak ada di bawaan", want)
		}
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

func TestRoadmapBawaanIsiLengkap(t *testing.T) {
	list, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range list {
		if len(r.Rules) == 0 {
			t.Fatalf("%s: rules kosong", r.ID)
		}
		if len(r.Sources) == 0 || len(r.Tips) == 0 {
			t.Fatalf("%s: sources/tips kosong", r.ID)
		}
		// Varian 5 bulan: 5 bulan x 4 minggu = 20 minggu.
		if len(r.Months) != 5 {
			t.Fatalf("%s: bulan = %d, want 5", r.ID, len(r.Months))
		}
		for _, m := range r.Months {
			if len(m.Weeks) != 4 {
				t.Fatalf("%s bulan %d: minggu = %d, want 4", r.ID, m.Index, len(m.Weeks))
			}
		}
		if got := r.TotalWeeks(); got != 20 {
			t.Fatalf("%s: total minggu = %d, want 20", r.ID, got)
		}
		// Mini project ada di minggu terakhir tiap blok proyek (7, 14, 20).
		for _, idx := range []int{7, 14, 20} {
			w, ok := r.FindWeek(idx)
			if !ok {
				t.Fatalf("%s: minggu %d hilang", r.ID, idx)
			}
			if w.MiniProject == "" {
				t.Fatalf("%s minggu %d: mini_project kosong", r.ID, idx)
			}
			if len(w.RequiredFeature) == 0 || len(w.PassCriteria) == 0 || len(w.Bonus) == 0 {
				t.Fatalf("%s minggu %d: fitur/syarat/bonus kosong", r.ID, idx)
			}
		}
		// Setiap minggu punya topik dan kompetensi yang bisa dicek.
		for i := 1; i <= r.TotalWeeks(); i++ {
			w, _ := r.FindWeek(i)
			if len(w.Topics) == 0 || len(w.HarusBisa) == 0 {
				t.Fatalf("%s minggu %d: topics/harus_bisa kosong", r.ID, i)
			}
		}
	}
}

// Panjang jalur bukan konstanta. Jalur 12 minggu (3 bulan) dan 20 minggu
// (5 bulan) sama-sama sah; yang wajib adalah indeks bulan dan minggu kontigu.
func TestValidatePanjangJalurBebas(t *testing.T) {
	buat := func(totalMinggu int) Roadmap {
		r := Roadmap{
			ID:             "ujicoba",
			Name:           "Jalur uji",
			Description:    "uji panjang jalur",
			Target:         "uji",
			SelfAssessment: SelfAssessment{Indicators: ValidIndicators},
		}
		for m := 1; (m-1)*4 < totalMinggu; m++ {
			mo := Month{Index: m, Title: "Bulan " + strconv.Itoa(m)}
			for w := (m-1)*4 + 1; w <= m*4 && w <= totalMinggu; w++ {
				mo.Weeks = append(mo.Weeks, Week{
					Index:     w,
					Title:     "Minggu " + strconv.Itoa(w),
					Topics:    []string{"Topik"},
					HarusBisa: []string{"Bisa"},
				})
			}
			r.Months = append(r.Months, mo)
		}
		return r
	}

	for _, n := range []int{12, 20} {
		r := buat(n)
		if got := r.TotalWeeks(); got != n {
			t.Fatalf("TotalWeeks = %d, want %d", got, n)
		}
		if got := r.TotalMonths(); got != (n+3)/4 {
			t.Fatalf("TotalMonths = %d, want %d", got, (n+3)/4)
		}
		if err := r.Validate(); err != nil {
			t.Fatalf("jalur %d minggu ditolak: %v", n, err)
		}
	}
}

// Indeks bulan yang bolong tetap ditolak.
func TestValidateBulanTidakKontigu(t *testing.T) {
	r := Roadmap{
		ID:             "ujicoba",
		Name:           "Jalur uji",
		Description:    "uji",
		Target:         "uji",
		SelfAssessment: SelfAssessment{Indicators: ValidIndicators},
	}
	r.Months = append(r.Months, Month{Index: 2, Title: "Bulan 2", Weeks: []Week{
		{Index: 1, Title: "Minggu 1", Topics: []string{"Topik"}, HarusBisa: []string{"Bisa"}},
	}})
	if err := r.Validate(); err == nil {
		t.Fatal("bulan 2 tanpa bulan 1 seharusnya ditolak")
	}
}

// Jumlah butir mengikuti dokumen sumber. Menyebar ulang ke 20 minggu tidak
// boleh menambah atau menghapus materi, jadi total ini dipakai sebagai penjaga.
func TestRoadmapBawaanJumlahButir(t *testing.T) {
	want := map[string]int{"swe-javascript": 58, "data-analyst": 48}
	list, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range list {
		n := 0
		for _, m := range r.Months {
			for _, w := range m.Weeks {
				if len(w.Tasks) > 0 {
					n += len(w.Tasks)
				} else {
					n += len(w.HarusBisa)
				}
			}
		}
		if n != want[r.ID] {
			t.Fatalf("%s: butir = %d, want %d", r.ID, n, want[r.ID])
		}
	}
}
