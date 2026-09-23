package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/HeyItWorked/riverroot/internal/pipeline"
	"github.com/HeyItWorked/riverroot/internal/store"
)

func newTestServer(t *testing.T) (*store.SQLiteStore, http.Handler) {
	t.Helper()
	s, err := store.NewSQLiteStore(filepath.Join(t.TempDir(), "builds.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	p := &pipeline.Pipeline{Name: "test", Steps: []pipeline.Step{{Name: "hello", Command: "echo hello"}}}
	return s, NewServer(s, p, t.TempDir()).SetupRouter()
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestListBuilds_ReturnsSavedBuilds(t *testing.T) {
	s, h := newTestServer(t)
	if _, err := s.Save(pipeline.BuildResult{Pipeline: "my-app"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	rec := get(t, h, "/builds")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /builds status = %d", rec.Code)
	}
	var builds []pipeline.BuildResult
	if err := json.NewDecoder(rec.Body).Decode(&builds); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if len(builds) != 1 || builds[0].Pipeline != "my-app" {
		t.Errorf("GET /builds = %+v", builds)
	}
}

func TestGetBuild_UnknownIDIs404(t *testing.T) {
	_, h := newTestServer(t)
	if rec := get(t, h, "/builds/no-such-build"); rec.Code != http.StatusNotFound {
		t.Errorf("GET /builds/no-such-build status = %d, want 404", rec.Code)
	}
}
