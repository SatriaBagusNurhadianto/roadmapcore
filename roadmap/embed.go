package roadmap

import (
	"embed"
	"fmt"
	"io/fs"
)

// roadmaps berisi satu berkas YAML per jalur.
//
// Konten roadmap diambil dari dokumen sumber (roadmap PDF Data Analyst dan
// SWE JavaScript) dan tidak dibuat-buat. Berkas README menjelaskan aturan
// penyusunannya.
//
//go:embed all:roadmaps
var embedded embed.FS

// LoadBuiltin memuat roadmap yang di-embed ke binary (offline).
func LoadBuiltin() ([]Roadmap, error) {
	sub, err := fs.Sub(embedded, "roadmaps")
	if err != nil {
		return nil, fmt.Errorf("embed roadmap: %w", err)
	}
	return LoadEmbedded(sub)
}
