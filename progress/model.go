// Package progress menyimpan progres roadmap dan state reminder dalam satu
// dokumen JSON bersama, dengan pemilik field yang ketat.
//
// Kontrak kepemilikan (uji memaksa aturan ini):
//
//	tracker.*   hanya ditulis Trace Learn
//	reminder.*  hanya ditulis Remind-Me-Learn
//
// Hash chain audit log yang dipakai di sini tanpa secret, sehingga hanya
// tamper-evident secara kasual — bukan bukti anti-manipulasi.
package progress

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SchemaVersion adalah versi schema progress.json dan settings.json.
const SchemaVersion = 1

// WeekStatus adalah status satu minggu.
type WeekStatus string

const (
	WeekNotStarted WeekStatus = "not_started"
	WeekInProgress WeekStatus = "in_progress"
	WeekWaiting    WeekStatus = "waiting"
	WeekDone       WeekStatus = "done"
)

// Valid memeriksa nilai status minggu.
func (s WeekStatus) Valid() bool {
	switch s {
	case WeekNotStarted, WeekInProgress, WeekWaiting, WeekDone:
		return true
	}
	return false
}

// ParseWeekStatus mengubah string menjadi WeekStatus.
func ParseWeekStatus(s string) (WeekStatus, error) {
	st := WeekStatus(s)
	if !st.Valid() {
		return "", fmt.Errorf("status minggu tidak dikenal: %q", s)
	}
	return st, nil
}

// MiniProjectStatus adalah status pengajuan mini project.
type MiniProjectStatus string

const (
	MPEmpty   MiniProjectStatus = "empty"
	MPWaiting MiniProjectStatus = "waiting"
	MPDone    MiniProjectStatus = "done"
)

// Valid memeriksa nilai status mini project.
func (s MiniProjectStatus) Valid() bool {
	switch s {
	case MPEmpty, MPWaiting, MPDone:
		return true
	}
	return false
}

// mpTransitions berisi transisi yang diizinkan menurut PRD Trace Learn FR-5.
var mpTransitions = map[MiniProjectStatus][]MiniProjectStatus{
	MPEmpty:   {MPWaiting},
	MPWaiting: {MPDone, MPEmpty},
	MPDone:    {MPWaiting, MPEmpty},
}

// CanTransition melaporkan apakah transisi status mini project diizinkan.
func (s MiniProjectStatus) CanTransition(to MiniProjectStatus) bool {
	for _, ok := range mpTransitions[s] {
		if ok == to {
			return true
		}
	}
	return false
}

// File adalah isi progress.json.
type File struct {
	SchemaVersion int       `json:"schema_version"`
	UpdatedAt     time.Time `json:"updated_at"`
	Tracker       *Tracker  `json:"tracker,omitempty"`
	Reminder      *Reminder `json:"reminder,omitempty"`
}

// Tracker adalah section milik Trace Learn.
type Tracker struct {
	ActivePath string                   `json:"active_path"`
	Paths      map[string]*PathProgress `json:"paths"`
}

// PathProgress adalah progres satu jalur.
type PathProgress struct {
	Weeks          map[string]*WeekState `json:"weeks"`
	MiniProject    MiniProject           `json:"mini_project"`
	SelfAssessment *SelfAssessment       `json:"self_assessment,omitempty"`
}

// WeekState adalah status satu minggu.
type WeekState struct {
	Status    WeekStatus `json:"status"`
	UpdatedAt time.Time  `json:"updated_at"`
	Note      string     `json:"note"`
}

// MiniProject adalah status pengajuan mini project.
type MiniProject struct {
	Status    MiniProjectStatus `json:"status"`
	URL       string            `json:"url"`
	CheckedAt *time.Time        `json:"checked_at,omitempty"`
	LastCheck *RepoCheck        `json:"last_check,omitempty"`
}

// RepoCheck menyimpan ringkasan metadata repository dari GitHub.
type RepoCheck struct {
	OwnerRepo   string     `json:"owner_repo"`
	Description string     `json:"description,omitempty"`
	HasReadme   bool       `json:"has_readme"`
	CommitCount int        `json:"commit_count"`
	HasHomepage bool       `json:"has_homepage"`
	Homepage    string     `json:"homepage,omitempty"`
	PushedAt    *time.Time `json:"pushed_at,omitempty"`
	CheckedAt   time.Time  `json:"checked_at"`
}

// SelfAssessment adalah refleksi akhir bulan.
type SelfAssessment struct {
	Month     int       `json:"month"`
	Level     string    `json:"level"`
	Note      string    `json:"note"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Reminder adalah section milik Remind-Me-Learn.
type Reminder struct {
	Checklist     map[string][]CheckItem `json:"checklist"`
	CustomTasks   []CustomTask           `json:"custom_tasks"`
	ReminderState ReminderState          `json:"reminder_state"`
	Focus         Focus                  `json:"focus"`
	Streak        Streak                 `json:"streak"`
	Settings      map[string]any         `json:"settings"`
}

// CheckItem adalah satu baris checklist harian.
type CheckItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	EstimateMin int    `json:"estimate_min,omitempty"`
	Done        bool   `json:"done"`
	Source      string `json:"source,omitempty"`
	CarriedFrom string `json:"carried_from,omitempty"`
	Hidden      bool   `json:"hidden,omitempty"`
	DoneAt      string `json:"done_at,omitempty"`
}

// CustomTask adalah tugas kustom pengguna.
type CustomTask struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	EstimateMin int    `json:"estimate_min,omitempty"`
	Date        string `json:"date"`
	DoneAt      string `json:"done_at,omitempty"`
	Done        bool   `json:"done"`
}

// ReminderState menyimpan state scheduler agar restart tidak menggandakan notifikasi.
type ReminderState struct {
	LastNotifiedDate string     `json:"last_notified_date"`
	SnoozeCount      int        `json:"snooze_count"`
	SnoozedUntil     *time.Time `json:"snoozed_until,omitempty"`
	LastNotifiedAt   *time.Time `json:"last_notified_at,omitempty"`
	LastError        string     `json:"last_error,omitempty"`
}

// Focus adalah timer fokus.
type Focus struct {
	StartedAt    *time.Time `json:"started_at,omitempty"`
	DurationMin  int        `json:"duration_min"`
	RemainingSec int        `json:"remaining_sec"`
	Running      bool       `json:"running"`
}

// Streak berisi ringkasan konsistensi.
type Streak struct {
	Count          int    `json:"count"`
	LastActiveDate string `json:"last_active_date"`
}

// Settings adalah isi settings.json.
type Settings struct {
	SchemaVersion int            `json:"schema_version"`
	UpdatedAt     time.Time      `json:"updated_at"`
	Tracker       map[string]any `json:"tracker"`
	Reminder      map[string]any `json:"reminder"`
}

// NewFile mengembalikan File kosong dengan section siap pakai.
func NewFile() *File {
	return &File{
		SchemaVersion: SchemaVersion,
		Tracker:       &Tracker{Paths: map[string]*PathProgress{}},
		Reminder: &Reminder{
			Checklist: map[string][]CheckItem{},
			Settings:  map[string]any{},
			Focus:     Focus{DurationMin: 25, RemainingSec: 25 * 60},
		},
	}
}

// NewSettings mengembalikan Settings kosong dengan section siap pakai.
func NewSettings() *Settings {
	return &Settings{
		SchemaVersion: SchemaVersion,
		Tracker:       map[string]any{},
		Reminder:      map[string]any{},
	}
}

// Normalize mengisi nilai default, menyeragamkan kunci minggu, dan memaksa
// semua timestamp internal memakai UTC.
func (f *File) Normalize() error {
	if f.Tracker == nil {
		f.Tracker = &Tracker{Paths: map[string]*PathProgress{}}
	}
	if f.Tracker.Paths == nil {
		f.Tracker.Paths = map[string]*PathProgress{}
	}
	for id, p := range f.Tracker.Paths {
		if p.Weeks == nil {
			p.Weeks = map[string]*WeekState{}
		}
		for key, w := range p.Weeks {
			if _, err := strconv.Atoi(key); err != nil {
				return fmt.Errorf("jalur %s: kunci minggu %q bukan angka", id, key)
			}
			if w.Status == "" {
				w.Status = WeekNotStarted
			}
			if w.UpdatedAt.IsZero() {
				w.UpdatedAt = f.UpdatedAt
			}
			w.UpdatedAt = w.UpdatedAt.UTC()
		}
		if p.MiniProject.Status == "" {
			p.MiniProject.Status = MPEmpty
		}
		if p.MiniProject.CheckedAt != nil {
			t := p.MiniProject.CheckedAt.UTC()
			p.MiniProject.CheckedAt = &t
		}
		if p.MiniProject.LastCheck != nil {
			if p.MiniProject.LastCheck.CheckedAt.IsZero() {
				p.MiniProject.LastCheck.CheckedAt = f.UpdatedAt
			}
			p.MiniProject.LastCheck.CheckedAt = p.MiniProject.LastCheck.CheckedAt.UTC()
			if p.MiniProject.LastCheck.PushedAt != nil {
				t := p.MiniProject.LastCheck.PushedAt.UTC()
				p.MiniProject.LastCheck.PushedAt = &t
			}
		}
		if p.SelfAssessment != nil {
			if p.SelfAssessment.UpdatedAt.IsZero() {
				p.SelfAssessment.UpdatedAt = f.UpdatedAt
			}
			p.SelfAssessment.UpdatedAt = p.SelfAssessment.UpdatedAt.UTC()
		}
	}
	if f.Reminder == nil {
		f.Reminder = NewFile().Reminder
	}
	if f.Reminder.Checklist == nil {
		f.Reminder.Checklist = map[string][]CheckItem{}
	}
	if f.Reminder.CustomTasks == nil {
		f.Reminder.CustomTasks = []CustomTask{}
	}
	if f.Reminder.Settings == nil {
		f.Reminder.Settings = map[string]any{}
	}
	if f.Reminder.Focus.DurationMin == 0 {
		f.Reminder.Focus.DurationMin = 25
	}
	if f.Reminder.Focus.StartedAt != nil {
		t := f.Reminder.Focus.StartedAt.UTC()
		f.Reminder.Focus.StartedAt = &t
	}
	if f.Reminder.ReminderState.SnoozedUntil != nil {
		t := f.Reminder.ReminderState.SnoozedUntil.UTC()
		f.Reminder.ReminderState.SnoozedUntil = &t
	}
	if f.Reminder.ReminderState.LastNotifiedAt != nil {
		t := f.Reminder.ReminderState.LastNotifiedAt.UTC()
		f.Reminder.ReminderState.LastNotifiedAt = &t
	}
	return nil
}

// Validate memeriksa keseluruhan dokumen.
func (f *File) Validate() error {
	if f.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema_version %d tidak didukung", f.SchemaVersion)
	}
	if f.Tracker == nil {
		return fmt.Errorf("section tracker hilang")
	}
	if f.Tracker.ActivePath == "" {
		return fmt.Errorf("tracker.active_path wajib diisi")
	}
	if _, ok := f.Tracker.Paths[f.Tracker.ActivePath]; !ok {
		return fmt.Errorf("jalur aktif %q tidak ada di tracker.paths", f.Tracker.ActivePath)
	}
	for id, p := range f.Tracker.Paths {
		for key, w := range p.Weeks {
			if !w.Status.Valid() {
				return fmt.Errorf("jalur %s minggu %s: status %q tidak valid", id, key, w.Status)
			}
		}
		if !p.MiniProject.Status.Valid() {
			return fmt.Errorf("jalur %s: status mini project %q tidak valid", id, p.MiniProject.Status)
		}
	}
	if f.Reminder != nil {
		for _, items := range f.Reminder.Checklist {
			for _, it := range items {
				if it.ID == "" || it.Title == "" {
					return fmt.Errorf("reminder.checklist: id dan title wajib diisi")
				}
			}
		}
		for _, ct := range f.Reminder.CustomTasks {
			if ct.ID == "" || ct.Title == "" || ct.Date == "" {
				return fmt.Errorf("reminder.custom_tasks: id, title, dan date wajib diisi")
			}
		}
	}
	return nil
}

// Path mengambil progres jalur, membuatnya bila belum ada.
func (t *Tracker) Path(id string) *PathProgress {
	if t.Paths == nil {
		t.Paths = map[string]*PathProgress{}
	}
	p, ok := t.Paths[id]
	if !ok {
		p = &PathProgress{Weeks: map[string]*WeekState{}, MiniProject: MiniProject{Status: MPEmpty}}
		t.Paths[id] = p
	}
	return p
}

// SetWeekStatus mengubah status satu minggu dengan timestamp.
func (p *PathProgress) SetWeekStatus(index int, st WeekStatus, now time.Time) error {
	if !st.Valid() {
		return fmt.Errorf("status minggu tidak dikenal: %q", st)
	}
	if p.Weeks == nil {
		p.Weeks = map[string]*WeekState{}
	}
	w, ok := p.Weeks[weekKey(index)]
	if !ok {
		w = &WeekState{Status: WeekNotStarted}
		p.Weeks[weekKey(index)] = w
	}
	w.Status = st
	w.UpdatedAt = now.UTC()
	return nil
}

// NoteOf mengembalikan catatan satu minggu (kosong bila belum ada).
func (p *PathProgress) NoteOf(index int) string {
	if w, ok := p.Weeks[weekKey(index)]; ok {
		return w.Note
	}
	return ""
}

// SetNote menyimpan catatan satu minggu. Indeks di luar 1..total ditolak.
//
// total adalah panjang jalur (rm.TotalWeeks()), bukan konstanta: jalur 3 bulan
// punya 12 minggu, jalur 5 bulan punya 20.
func (p *PathProgress) SetNote(index, total int, note string) error {
	if total < 1 {
		total = 1
	}
	if index < 1 || index > total {
		return fmt.Errorf("minggu %d di luar rentang 1..%d", index, total)
	}
	if p.Weeks == nil {
		p.Weeks = map[string]*WeekState{}
	}
	key := weekKey(index)
	w, ok := p.Weeks[key]
	if !ok {
		w = &WeekState{Status: WeekNotStarted}
		p.Weeks[key] = w
	}
	w.Note = note
	return nil
}

// WeekStatusOf mengembalikan status minggu, default not_started.
func (p *PathProgress) WeekStatusOf(index int) WeekStatus {
	if w, ok := p.Weeks[weekKey(index)]; ok && w.Status.Valid() {
		return w.Status
	}
	return WeekNotStarted
}

// ActiveWeekUpTo mengembalikan minggu pertama (1..total) yang belum done.
func (p *PathProgress) ActiveWeekUpTo(total int) int {
	if total < 1 {
		total = 1
	}
	for i := 1; i <= total; i++ {
		if p.WeekStatusOf(i) != WeekDone {
			return i
		}
	}
	return total
}

// DoneWeeksUpTo menghitung minggu berstatus done pada 1..total.
func (p *PathProgress) DoneWeeksUpTo(total int) int {
	if total < 1 {
		return 0
	}
	n := 0
	for i := 1; i <= total; i++ {
		if p.WeekStatusOf(i) == WeekDone {
			n++
		}
	}
	return n
}

// Transition memvalidasi dan menerapkan transisi status mini project.
func (m *MiniProject) Transition(to MiniProjectStatus, now time.Time) error {
	if !to.Valid() {
		return fmt.Errorf("status mini project tidak dikenal: %q", to)
	}
	if !m.Status.CanTransition(to) {
		return fmt.Errorf("transisi mini project %s -> %s tidak diizinkan", m.Status, to)
	}
	m.Status = to
	utc := now.UTC()
	m.CheckedAt = &utc
	return nil
}

func weekKey(index int) string {
	return strconv.Itoa(index)
}

// Validate memeriksa settings.
func (s *Settings) Validate() error {
	if s.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema_version %d tidak didukung", s.SchemaVersion)
	}
	return nil
}

// Normalize mengisi section kosong pada settings.
func (s *Settings) Normalize() {
	if s.Tracker == nil {
		s.Tracker = map[string]any{}
	}
	if s.Reminder == nil {
		s.Reminder = map[string]any{}
	}
}

// ValidateLevel memeriksa nilai level self-assessment.
func ValidateLevel(level string) error {
	switch strings.ToLower(level) {
	case "pemula", "menengah_awal", "di_atas_rata_rata":
		return nil
	}
	return fmt.Errorf("level self-assessment tidak dikenal: %q", level)
}
