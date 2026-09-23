package store

import (
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/HeyItWorked/riverroot/internal/pipeline"
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

func TestSQLiteStore_GetFillsIDAndCreatedAt(t *testing.T) {
	s, err := NewSQLiteStore(filepath.Join(t.TempDir(), "builds.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	id, err := s.Save(sampleBuild("my-app", false))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get(%q): %v", id, err)
	}
	if got.ID != id {
		t.Errorf("Get(%q).ID = %q", id, got.ID)
	}
	if got.CreatedAt == "" {
		t.Errorf("Get(%q).CreatedAt is empty", id)
	}
}

func TestSQLiteStore_GetUnknownIDReturnsErrNoRows(t *testing.T) {
	s, err := NewSQLiteStore(filepath.Join(t.TempDir(), "builds.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	got, err := s.Get("no-such-build")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("Get(unknown id) err = %v", err)
	}
	if !reflect.DeepEqual(got, pipeline.BuildResult{}) {
		t.Errorf("Get(unknown id) = %+v", got)
	}
}

func TestSQLiteStore_PersistsAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "builds.db")
	want := sampleBuild("my-app", true)
	first, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	id, err := first.Save(want)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	first.Close()
	second, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("reopening SQLiteStore: %v", err)
	}
	t.Cleanup(func() { second.Close() })
	got, err := second.Get(id)
	if err != nil {
		t.Fatalf("Get(%q) after reopen: %v", id, err)
	}
	if !reflect.DeepEqual(storedShape(got), storedShape(want)) {
		t.Errorf("Get(%q) after reopen", id)
	}
}
