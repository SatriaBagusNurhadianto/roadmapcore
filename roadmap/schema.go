// Package roadmap memuat dan memvalidasi data jalur belajar (roadmap).
//
// Satu jalur = satu dokumen YAML. Konten roadmap wajib berasal dari dokumen
// sumber; paket ini hanya menyediakan skema dan validator, tidak menyimpan
// materi apa pun.
package roadmap

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ExpectedWeeks adalah jumlah minggu dalam satu jalur pada format v1.
const ExpectedWeeks = 12

// ValidIndicators adalah daftar indikator self-assessment yang diizinkan.
var ValidIndicators = []string{"pemula", "menengah_awal", "di_atas_rata_rata"}

var idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Roadmap adalah satu jalur belajar lengkap.
type Roadmap struct {
	ID             string         `yaml:"id" json:"id"`
	Name           string         `yaml:"name" json:"name"`
	Description    string         `yaml:"description" json:"description"`
	Target         string         `yaml:"target" json:"target"`
	Rules          []string       `yaml:"rules" json:"rules"`
	Months         []Month        `yaml:"months" json:"months"`
	SelfAssessment SelfAssessment `yaml:"self_assessment" json:"self_assessment"`
}

// Month mengelompokkan minggu-minggu dalam satu bulan.
type Month struct {
	Index int    `yaml:"index" json:"index"`
	Title string `yaml:"title" json:"title"`
	Weeks []Week `yaml:"weeks" json:"weeks"`
}

// Week adalah materi satu minggu.
type Week struct {
	Index        int      `yaml:"index" json:"index"`
	Title        string   `yaml:"title" json:"title"`
	Topics       []string `yaml:"topics" json:"topics"`
	HarusBisa    []string `yaml:"harus_bisa" json:"harus_bisa"`
	Tasks        []Task   `yaml:"tasks" json:"tasks"`
	MiniProject  string   `yaml:"mini_project" json:"mini_project"`
	PassCriteria []string `yaml:"pass_criteria" json:"pass_criteria"`
	Bonus        []string `yaml:"bonus" json:"bonus"`
	Sources      []string `yaml:"sources" json:"sources"`
	Tips         []string `yaml:"tips" json:"tips"`
}

// Task adalah satu tugas dalam minggu.
type Task struct {
	Title       string `yaml:"title" json:"title"`
	EstimateMin int    `yaml:"estimate_min" json:"estimate_min"`
	Optional    bool   `yaml:"optional" json:"optional"`
}

// SelfAssessment memuat indikator refleksi akhir bulan.
type SelfAssessment struct {
	Indicators []string `yaml:"indicators" json:"indicators"`
}

// Validate memeriksa invarian format v1. Mengembalikan error pertama yang
// ditemukan, bukan mengumpulkan semua error, agar pesan tetap mudah ditindaklanjuti.
func (r Roadmap) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("roadmap: id wajib diisi")
	}
	if !idPattern.MatchString(r.ID) {
		return fmt.Errorf("roadmap %s: id harus kebab-case kecil, dapatkan %q", r.ID, r.ID)
	}
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("roadmap %s: name wajib diisi", r.ID)
	}
	if len(r.Months) == 0 {
		return fmt.Errorf("roadmap %s: months wajib diisi", r.ID)
	}

	seenMonth := map[int]bool{}
	seenWeek := map[int]bool{}
	total := 0
	for _, m := range r.Months {
		if m.Index <= 0 {
			return fmt.Errorf("roadmap %s: month index harus > 0, dapat %d", r.ID, m.Index)
		}
		if seenMonth[m.Index] {
			return fmt.Errorf("roadmap %s: month index %d duplikat", r.ID, m.Index)
		}
		seenMonth[m.Index] = true
		if strings.TrimSpace(m.Title) == "" {
			return fmt.Errorf("roadmap %s: month %d title wajib diisi", r.ID, m.Index)
		}
		for _, w := range m.Weeks {
			if w.Index <= 0 {
				return fmt.Errorf("roadmap %s: week index harus > 0, dapat %d", r.ID, w.Index)
			}
			if seenWeek[w.Index] {
				return fmt.Errorf("roadmap %s: week index %d duplikat", r.ID, w.Index)
			}
			seenWeek[w.Index] = true
			total++
			if err := validateWeek(r.ID, w); err != nil {
				return err
			}
		}
	}

	if total != ExpectedWeeks {
		return fmt.Errorf("roadmap %s: total minggu %d, wajib %d", r.ID, total, ExpectedWeeks)
	}
	// Indeks minggu harus kontigu 1..ExpectedWeeks.
	for i := 1; i <= ExpectedWeeks; i++ {
		if !seenWeek[i] {
			return fmt.Errorf("roadmap %s: week index %d hilang (indeks harus kontigu 1..%d)", r.ID, i, ExpectedWeeks)
		}
	}
	return validateIndicators(r)
}

func validateWeek(roadmapID string, w Week) error {
	if strings.TrimSpace(w.Title) == "" {
		return fmt.Errorf("roadmap %s: week %d title wajib diisi", roadmapID, w.Index)
	}
	if len(w.Topics) == 0 {
		return fmt.Errorf("roadmap %s: week %d topics wajib diisi", roadmapID, w.Index)
	}
	for _, t := range w.Tasks {
		if strings.TrimSpace(t.Title) == "" {
			return fmt.Errorf("roadmap %s: week %d task title wajib diisi", roadmapID, w.Index)
		}
		if t.EstimateMin <= 0 {
			return fmt.Errorf("roadmap %s: week %d task %q estimate_min harus > 0", roadmapID, w.Index, t.Title)
		}
	}
	return nil
}

func validateIndicators(r Roadmap) error {
	got := append([]string(nil), r.SelfAssessment.Indicators...)
	want := append([]string(nil), ValidIndicators...)
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		return fmt.Errorf("roadmap %s: self_assessment.indicators wajib %v, dapat %v", r.ID, ValidIndicators, r.SelfAssessment.Indicators)
	}
	for i := range got {
		if got[i] != want[i] {
			return fmt.Errorf("roadmap %s: self_assessment.indicators wajib %v, dapat %v", r.ID, ValidIndicators, r.SelfAssessment.Indicators)
		}
	}
	return nil
}

// WeekIndices mengembalikan daftar indeks minggu terurut.
func (r Roadmap) WeekIndices() []int {
	idx := make([]int, 0, ExpectedWeeks)
	for _, m := range r.Months {
		for _, w := range m.Weeks {
			idx = append(idx, w.Index)
		}
	}
	sort.Ints(idx)
	return idx
}

// FindWeek mengembalikan minggu berdasarkan indeks, atau false bila tidak ada.
func (r Roadmap) FindWeek(index int) (Week, bool) {
	for _, m := range r.Months {
		for _, w := range m.Weeks {
			if w.Index == index {
				return w, true
			}
		}
	}
	return Week{}, false
}
