package progress

import "encoding/json"

// MarshalFile mengubah File menjadi JSON rapi dengan indentasi.
// Dipakai ekspor dan test; bukan jalur tulis utama.
func MarshalFile(f *File) ([]byte, error) {
	if err := f.Normalize(); err != nil {
		return nil, err
	}
	return json.MarshalIndent(f, "", "  ")
}

// UnmarshalFile membaca JSON File lalu menormalkannya.
func UnmarshalFile(data []byte) (*File, error) {
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if f.SchemaVersion > SchemaVersion {
		return nil, ErrSchemaTooNew
	}
	if f.SchemaVersion == 0 {
		f = *NewFile()
	}
	if err := f.Normalize(); err != nil {
		return nil, err
	}
	return &f, nil
}

// MarshalSettings mengubah Settings menjadi JSON rapi.
func MarshalSettings(s *Settings) ([]byte, error) {
	s.Normalize()
	return json.MarshalIndent(s, "", "  ")
}

// UnmarshalSettings membaca JSON Settings lalu menormalkannya.
func UnmarshalSettings(data []byte) (*Settings, error) {
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.SchemaVersion > SchemaVersion {
		return nil, ErrSchemaTooNew
	}
	if s.SchemaVersion == 0 {
		s = *NewSettings()
	}
	s.Normalize()
	return &s, nil
}
