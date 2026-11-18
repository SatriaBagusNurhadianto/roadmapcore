# Roadmap bawaan

Letakkan satu berkas `.yaml` per jalur di direktori ini.

Konten roadmap **harus** disalin dari dokumen sumber (roadmap PDF Data Analyst
dan SWE JavaScript). Jangan mengarang, menerjemahkan bebas, atau menghapus
materi tanpa persetujuan. Skema dan aturan validasinya ada di
`../docs/FORMAT.md`.

## Isi bawaan

| Berkas | Sumber | Minggu |
|---|---|---|
| `data-analyst.yaml` | "Roadmap Data Analyst" (Mini E-Book) | 20 (5 bulan) |
| `swe-javascript.yaml` | "Roadmap Software Engineer (JavaScript)" (Mini E-Book) | 20 (5 bulan) |

## Penjadwalan 5 bulan

Dokumen sumber memadatkan tiap jalur dalam 12 minggu (3 bulan). Berkas bawaan
menyebar ulang **materi yang sama** ke 20 minggu (5 bulan x 4 minggu) supaya
beban per minggu turun (swe-javascript 4,8 -> 2,9 butir/minggu; data-analyst
4,0 -> 2,4). Tidak ada materi yang ditambah atau dihapus; urutan topik dan isi
butir mengikuti dokumen sumber.

Tiga mini project dipakai sebagai tolok ukur blok, masing-masing berjarak 42-49
hari (kurang lebih 1,5 bulan):

| Jalur | Proyek 1 | Proyek 2 | Proyek 3 |
|---|---|---|---|
| swe-javascript | akhir minggu 7 | akhir minggu 14 | akhir minggu 20 |
| data-analyst | akhir minggu 7 | akhir minggu 14 | akhir minggu 20 |

Panjang jalur bukan konstanta: `Roadmap.TotalWeeks()` yang dipakai sebagai
batas indeks, bukan `12`.

## Aturan penyusunan

- `topics` = kolom "Topik" tabel dokumen sumber.
- `harus_bisa` = kolom "Yang harus bisa", dipecah pada koma di luar tanda
  kurung. Sel yang berisi titik dua diperlakukan sebagai satu butir utuh.
- `mini_project`, `required_features`, `pass_criteria`, `bonus` = bagian
  "Mini Project Bulanan", diletakkan pada minggu terakhir tiap blok proyek.
- `sources` dan `tips` = bagian 5 dokumen, dipindah ke level roadmap.
- Tabel "Cek Progres" tidak punya field di skema v1; disimpan sebagai komentar
  di awal berkas agar tidak hilang.
