package progress

import "fmt"

// migration adalah satu langkah migrasi schema.
type migration struct {
	from  int
	to    int
	apply func(*File) error
}

// migrations berurutan dari versi paling lama. Tambahkan entri baru di akhir,
// jangan pernah mengubah entri yang sudah ada.
var migrations = []migration{
	{from: 0, to: 1, apply: func(f *File) error { return nil }},
}

// migrateVersion menaikkan nomor versi dan menolak versi yang lebih baru.
func migrateVersion(v int) (int, error) {
	if v > SchemaVersion {
		return 0, fmt.Errorf("%w: %d > %d", ErrSchemaTooNew, v, SchemaVersion)
	}
	for _, m := range migrations {
		if v == m.from {
			v = m.to
		}
	}
	if v != SchemaVersion {
		return 0, fmt.Errorf("schema %d tidak dapat dimigrasi ke %d", v, SchemaVersion)
	}
	return v, nil
}

// Migrate menaikkan schema File ke SchemaVersion.
func Migrate(f *File) error {
	v, err := migrateVersion(f.SchemaVersion)
	if err != nil {
		return err
	}
	for _, m := range migrations {
		if m.from == f.SchemaVersion && m.apply != nil {
			if err := m.apply(f); err != nil {
				return fmt.Errorf("migrasi schema %d -> %d: %w", m.from, m.to, err)
			}
		}
	}
	f.SchemaVersion = v
	return nil
}

// MigrateSettings menaikkan schema Settings ke SchemaVersion.
func MigrateSettings(s *Settings) error {
	v, err := migrateVersion(s.SchemaVersion)
	if err != nil {
		return err
	}
	s.SchemaVersion = v
	return nil
}
