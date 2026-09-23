package store

import (
	"path/filepath"
	"testing"
)

func TestSQLiteStore_SaveWritesBuildAndSteps(t *testing.T) {
	s, err := NewSQLiteStore(filepath.Join(t.TempDir(), "builds.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	id, err := s.Save(sampleBuild("my-app", true))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	var builds, steps int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM builds WHERE id = ?", id).Scan(&builds); err != nil {
		t.Fatalf("counting builds: %v", err)
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM steps WHERE build_id = ?", id).Scan(&steps); err != nil {
		t.Fatalf("counting steps: %v", err)
	}
	if builds != 1 || steps != 2 {
		t.Errorf("Save(%q) wrote %d builds and %d steps, want 1 and 2", id, builds, steps)
	}
}
