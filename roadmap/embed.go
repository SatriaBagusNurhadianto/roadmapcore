package roadmap

import "embed"

// roadmaps berisi satu berkas YAML per jalur.
//
// Saat ini belum ada berkas roadmap asli di dalam repositori: konten roadmap
// wajib diambil dari dokumen sumber (PDF) dan tidak boleh dibuat-buat.
// Berkas README ini sengaja ada agar direktori embed tidak kosong.
//
//go:embed all:roadmaps
var embedded embed.FS

// LoadBuiltin memuat roadmap yang di-embed ke binary (offline).
func LoadBuiltin() ([]Roadmap, error) {
	return LoadEmbedded(embedded)
}
