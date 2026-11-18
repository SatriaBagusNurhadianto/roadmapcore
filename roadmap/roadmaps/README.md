# Roadmap bawaan

Letakkan satu berkas `.yaml` per jalur di direktori ini.

Konten roadmap **harus** disalin dari dokumen sumber (roadmap PDF Data Analyst
dan SWE JavaScript). Jangan mengarang, menerjemahkan bebas, atau menghapus
materi tanpa persetujuan. Skema dan aturan validasinya ada di
`../docs/FORMAT.md`.

## Isi bawaan

| Berkas | Sumber |
|---|---|
| `data-analyst.yaml` | "Roadmap Data Analyst" (Mini E-Book) |
| `swe-javascript.yaml` | "Roadmap Software Engineer (JavaScript)" (Mini E-Book) |

## Aturan penyusunan

- `topics` = kolom "Topik" tabel 12 minggu.
- `harus_bisa` = kolom "Yang harus bisa", dipecah pada koma di luar tanda
  kurung. Sel yang berisi titik dua diperlakukan sebagai satu butir utuh.
- `mini_project`, `required_features`, `pass_criteria`, `bonus` = bagian
  "Mini Project Bulanan", diletakkan pada minggu terakhir bulan bersangkutan.
- `sources` dan `tips` = bagian 5 dokumen, dipindah ke level roadmap.
- Tabel "Cek Progres" tidak punya field di skema v1; disimpan sebagai komentar
  di awal berkas agar tidak hilang.
