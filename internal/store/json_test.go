package store

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/HeyItWorked/riverroot/internal/pipeline"
)

func sampleBuild(name string, failed bool) pipeline.BuildResult {
	return pipeline.BuildResult{
		Pipeline: name,
		Failed:   failed,
		Steps: []pipeline.StepResult{
			{Name: "lint", ExitCode: 0, Stdout: "checking style...\nno issues found\n"},
			{Name: "test", ExitCode: 1, Stdout: "--- FAIL: TestUserAuth\n", Stderr: "FAIL\n"},
		},
	}
}

func storedShape(r pipeline.BuildResult) pipeline.BuildResult {
	r.ID = ""
	r.CreatedAt = ""
	return r
}

func TestJSONStore_GetRoundTrip(t *testing.T) {
	s, err := NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	want := sampleBuild("my-app", true)
	id, err := s.Save(want)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get(%q): %v", id, err)
	}
	if !reflect.DeepEqual(storedShape(got), storedShape(want)) {
		t.Errorf("Get(%q)", id)
	}
}

func TestJSONStore_SaveWritesFileNamedByID(t *testing.T) {
	dir := t.TempDir()
	s, err := NewJSONStore(dir)
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	id, err := s.Save(sampleBuild("my-app", false))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	path := filepath.Join(dir, id+".json")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("stat %q: %v", path, err)
	}
}

func TestJSONStore_GetUnknownIDReturnsNotExist(t *testing.T) {
	s, err := NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	got, err := s.Get("no-such-build")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Get(unknown id) err = %v", err)
	}
	if !reflect.DeepEqual(got, pipeline.BuildResult{}) {
		t.Errorf("Get(unknown id) = %+v", got)
	}
}

func TestJSONStore_ListIgnoresUnusableFiles(t *testing.T) {
	dir := t.TempDir()
	s, err := NewJSONStore(dir)
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	want := sampleBuild("my-app", false)
	if _, err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not a build"), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	got, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("List returned %d builds, want 1", len(got))
	}
	if !reflect.DeepEqual(storedShape(got[0]), storedShape(want)) {
		t.Errorf("List()")
	}
}
