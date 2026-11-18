# Format data Roadmap Suite

Dokumen ini menjelaskan format berkas yang dikelola `roadmapcore`. Satu direktori
data dipakai dua aplikasi (Trace Learn dan Remind-Me-Learn); pemilik field di bawah
adalah kontrak yang dipaksakan oleh API dan diuji.

## Lokasi direktori

| OS | Path |
|---|---|
| Windows | `%AppData%/roadmap-suite/` |
| Linux | `$XDG_CONFIG_HOME/roadmap-suite/` atau `~/.config/roadmap-suite/` |
| macOS | `~/Library/Application Support/roadmap-suite/` |

## Berkas

| Berkas | Isi |
|---|---|
| `progress.json` | progres roadmap (tracker) + checklist/streak/fokus (reminder) |
| `progress.json.bak` | salinan versi sebelumnya, diputar tiap write berhasil |
| `settings.json` | preferensi, dipisah per section seperti progress |
| `audit.jsonl` | log append-only + hash chain |
| `progress.lock`, `settings.lock`, `audit.lock`, `daemon.lock` | lock file, tidak pernah dihapus/di-rename |

## Kepemilikan field

| Path | Penulis | Pembaca |
|---|---|---|
| `schema_version`, `updated_at` | core (otomatis) | kedua aplikasi |
| `tracker.*` | Trace Learn | Trace Learn tulis; Reminder baca |
| `reminder.*` | Remind-Me-Learn | Reminder tulis; Trace Learn baca |

Melanggar kepemilikan berarti ada bug. Reminder tidak boleh mengubah status minggu
atau status mini project.

## progress.json

```json
{
  "schema_version": 1,
  "updated_at": "2026-11-17T14:03:22Z",
  "tracker": {
    "active_path": "data-analyst",
    "paths": {
      "data-analyst": {
        "weeks": {
          "1": { "status": "done", "updated_at": "2026-11-10T09:12:00Z", "note": "" }
        },
        "mini_project": {
          "status": "empty",
          "url": "",
          "checked_at": "2026-11-17T13:00:00Z",
          "last_check": {
            "owner_repo": "user/repo",
            "description": "…",
            "has_readme": true,
            "commit_count": 12,
            "has_homepage": false,
            "homepage": "",
            "pushed_at": "2026-11-16T08:00:00Z",
            "checked_at": "2026-11-17T13:00:00Z"
          }
        },
        "self_assessment": { "month": 1, "level": "menengah_awal", "note": "", "updated_at": "…" }
      }
    }
  },
  "reminder": {
    "checklist": {
      "2026-11-17": [
        {
          "id": "t-01",
          "title": "…",
          "estimate_min": 30,
          "done": false,
          "source": "roadmap:data-analyst:week:2:task:1",
          "carried_from": ""
        }
      ]
    },
    "custom_tasks": [],
    "reminder_state": {
      "last_notified_date": "2026-11-17",
      "snooze_count": 0,
      "snoozed_until": null
    },
    "focus": { "started_at": null, "duration_min": 25, "remaining_sec": 1500, "running": false },
    "streak": { "count": 3, "last_active_date": "2026-11-16" },
    "settings": {}
  }
}
```

Enum:

- status minggu: `not_started`, `in_progress`, `waiting`, `done`
- status mini project: `empty`, `waiting`, `done`
- transisi mini project yang diizinkan: `empty→waiting`, `waiting→done`,
  `waiting→empty`, `done→waiting`, `done→empty`
- level self-assessment: `pemula`, `menengah_awal`, `di_atas_rata_rata`

Timestamp internal memakai RFC3339 UTC. Tanggal untuk checklist/streak memakai
tanggal lokal `YYYY-MM-DD`.

## audit.jsonl

Satu JSON per baris, append-only:

```json
{"seq":1,"id":"019c…","ts":"2026-11-17T14:03:22Z","app":"tracker","event":"week.status","summary":"week 2: in_progress -> done","prev":"","hash":"<sha256>"}
```

Rumus rantai:

```
hash_n = sha256( hash_{n-1} || canonical_json(payload_tanpa_hash) )
hash_0    = "" (kosong)
```

`Validate` mengembalikan `seq` pertama yang tidak cocok, atau `0` bila seluruhnya
valid. Bila ada yang tidak cocok, aplikasi menampilkan peringatan dan **tidak**
memperbaiki log secara otomatis.

**Batas jujur:** tanpa secret, hash chain ini hanya tamper-evident secara kasual.
Pemilik perangkat yang menguasai berkas bisa mengubah data lalu menghitung ulang
hash. Ini bukan bukti anti-manipulasi.

## Locking

- Lock pada berkas terpisah (`progress.lock`, dst.), **bukan** pada berkas data.
  Atomic replace mengganti inode berkas data, sehingga lock di sana tidak
  melindungi berkas baru.
- Windows: `windows.LockFileEx` eksklusif non-blocking. Unix: `flock(LOCK_EX|LOCK_NB)`.
- Retry maksimal ±2 detik dengan jeda acak 50–100 ms, lalu error yang dapat
  ditindaklanjuti. Tanpa jeda acak semua penunggu bangun bersamaan dan throughput
  runtuh.
- Lock hanya untuk critical section baca→ubah→tulis.

## Atomic write

1. Tulis `progress.json.tmp-<pid>-<rand>` di direktori yang sama, mode `0600`.
2. `Sync()`, `Close()`.
3. Putar `progress.json` lama ke `progress.json.bak`.
4. `os.Rename(tmp, progress.json)`.
5. `fsync` direktori (best-effort).

## Migrasi dan kerusakan

- `schema_version` < current → migrasi eksplisit, backup dulu.
- `schema_version` > current → read-only, menolak tulis, minta upgrade.
- JSON rusak → berkas rusak **dipertahankan**, `.bak` dicoba, pengguna diberi pilihan
  export/restore. Tidak ada penghapusan otomatis.
