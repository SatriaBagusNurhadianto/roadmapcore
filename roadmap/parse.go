package roadmap

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parse mengubah satu dokumen YAML menjadi Roadmap yang sudah divalidasi.
// KnownFields(true) dipakai agar field yang salah ketik gagal terbaca,
// bukan diam-diam diabaikan.
func Parse(data []byte) (Roadmap, error) {
	var r Roadmap
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&r); err != nil {
		return Roadmap{}, fmt.Errorf("parse roadmap: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return Roadmap{}, fmt.Errorf("parse roadmap: dokumen berisi lebih dari satu YAML")
	}
	if err := r.Validate(); err != nil {
		return Roadmap{}, err
	}
	return r, nil
}

// ParseFile membaca dan mengurai satu berkas YAML.
func ParseFile(name string) (Roadmap, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return Roadmap{}, fmt.Errorf("baca %s: %w", name, err)
	}
	r, err := Parse(data)
	if err != nil {
		return Roadmap{}, fmt.Errorf("%s: %w", name, err)
	}
	return r, nil
}

// LoadDir memuat semua berkas .yaml di sebuah direktori, diurutkan nama berkas.
// Direktori yang belum ada menghasilkan error fs.ErrNotExist.
func LoadDir(dir string) ([]Roadmap, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("baca direktori %s: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	out := make([]Roadmap, 0, len(names))
	for _, n := range names {
		r, err := ParseFile(path.Join(dir, n))
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

// LoadEmbedded memuat roadmap yang di-embed ke binary.
func LoadEmbedded(fsys fs.FS) ([]Roadmap, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("baca embed roadmap: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	// Tanpa berkas .yaml bukan error: aplikasi tetap bisa memakai roadmap
	// yang diletakkan pengguna di direktori data.
	if len(names) == 0 {
		return nil, nil
	}

	out := make([]Roadmap, 0, len(names))
	for _, n := range names {
		data, err := fs.ReadFile(fsys, n)
		if err != nil {
			return nil, fmt.Errorf("baca embed %s: %w", n, err)
		}
		r, err := Parse(data)
		if err != nil {
			return nil, fmt.Errorf("embed %s: %w", n, err)
		}
		out = append(out, r)
	}
	return out, nil
}
