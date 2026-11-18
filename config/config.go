// Package config menentukan lokasi direktori data bersama Roadmap Suite.
// Dipakai dua aplikasi (Trace Learn dan Remind-Me-Learn) supaya path data
// tidak pernah berbeda di antara keduanya.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// DirName adalah nama direktori di dalam direktori konfigurasi pengguna.
const DirName = "roadmap-suite"

// Paths memuat semua path yang dipakai dua aplikasi.
type Paths struct {
	Dir      string
	Progress string
	Settings string
	Audit    string
	Roadmaps string
	Cache    string
}

// Resolve menentukan direktori data. Override (flag --data-dir) dipakai bila
// tidak kosong; bila kosong ikut konvensi OS lewat os.UserConfigDir:
// Windows %AppData%, Linux $XDG_CONFIG_HOME atau ~/.config,
// macOS ~/Library/Application Support.
func Resolve(override string) (*Paths, error) {
	base := override
	if base == "" {
		d, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("tentukan direktori konfigurasi: %w", err)
		}
		base = d
	}
	dir := filepath.Join(base, DirName)
	p := &Paths{
		Dir:      dir,
		Progress: filepath.Join(dir, "progress.json"),
		Settings: filepath.Join(dir, "settings.json"),
		Audit:    filepath.Join(dir, "audit.jsonl"),
		Roadmaps: filepath.Join(dir, "roadmaps"),
		Cache:    filepath.Join(dir, "cache"),
	}
	return p, nil
}

// Ensure membuat direktori data bila belum ada.
func Ensure(p *Paths) error {
	for _, dir := range []string{p.Dir, p.Roadmaps, p.Cache} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("buat direktori %s: %w", dir, err)
		}
	}
	return nil
}
