package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveOverride(t *testing.T) {
	p, err := Resolve(`C:/data`)
	if err != nil {
		t.Fatal(err)
	}
	if p.Dir != filepath.Join(`C:/data`, DirName) {
		t.Fatalf("dir = %q", p.Dir)
	}
	if p.Progress != filepath.Join(p.Dir, "progress.json") {
		t.Fatalf("progress = %q", p.Progress)
	}
	if p.Cache != filepath.Join(p.Dir, "cache") {
		t.Fatalf("cache = %q", p.Cache)
	}
}

func TestEnsureMembuatDirektori(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Permission bit 0700 di Windows tidak dibaca OS; pemeriksaan cukup
		// memastikan direktori dan subdirektornya ada.
		t.Log("windows: pemeriksaan permission dilewati")
	}
	p, err := Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := Ensure(p); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{p.Dir, p.Roadmaps, p.Cache} {
		if _, err := os.Stat(d); err != nil {
			t.Fatalf("%s: %v", d, err)
		}
	}
}
