# roadmapcore

Module Go bersama untuk **Roadmap Suite** (Trace Learn + Remind-Me-Learn).

Menyediakan:

- `roadmap` — skema + parser + validator roadmap YAML (di-embed ke binary).
  Panjang jalur bebas (>= 1 minggu); `Roadmap.TotalWeeks()` dipakai sebagai
  batas indeks, bukan angka keras.
- `progress` — model progres, cross-process lock, atomic write, backup, migrasi,
  dan audit log hash-chain.

## Status

Model data, storage, locking, dan audit sudah diimplementasikan dan diuji.

Konten roadmap **sudah ada** dan diambil dari dokumen sumber: lihat
`roadmap/roadmaps/data-analyst.yaml` (Roadmap Data Analyst) dan
`roadmap/roadmaps/swe-javascript.yaml` (Roadmap Software Engineer JavaScript).
Isi roadmap tidak dibuat-buat; aturan penyusunannya ada di
`roadmap/roadmaps/README.md`.

## Pemakaian

```go
r, err := progress.Open(dir)
err = r.UpdateProgress(ctx, func(f *progress.File) error {
    p := f.Tracker.Path("data-analyst")
    return p.SetWeekStatus(2, progress.WeekDone, time.Now())
})
```

`UpdateProgress` mengambil lock, membaca versi terbaru, menjalankan `fn`, lalu
menulis atomik. Jangan pernah menulis berkas data langsung dari aplikasi.

## Kontrak kepemilikan

`tracker.*` hanya ditulis Trace Learn; `reminder.*` hanya ditulis Remind-Me-Learn.
Detail format: [`docs/FORMAT.md`](docs/FORMAT.md).

## Pengujian

```bash
gofmt -l .          # harus kosong
go vet ./...
go test ./... -count=1
go test ./... -race -count=1   # butuh C compiler
```

## Lisensi

Apache-2.0 — lihat [LICENSE](LICENSE).
