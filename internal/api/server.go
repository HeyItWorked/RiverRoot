// Package api is the HTTP layer for riverroot.
// It exposes build data over REST and serves the embedded dashboard.
//
// Reused from: babel-shelf/go-bookshelf (handlers.go + main.go)
// Same pattern: handler reads request → talks to store → writes JSON response.
// respondJSON and setupRouter are near-identical copy-paste.
//
// What changed:
//   - Store is a struct field, not a package-level var (riverroot already uses struct-based stores)
//   - IDs are strings (UUIDs) not ints — no strconv.Atoi needed
//   - No Update/Delete — CI builds are immutable
//
// ~60% reused structure, ~40% riverroot-specific changes.
package api

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"

	"github.com/HeyItWorked/riverroot/internal/pipeline"
	"github.com/HeyItWorked/riverroot/internal/store"
)

// embeds static/ directory into the binary at compile time — no external files needed at runtime
//
//go:embed static
var staticFiles embed.FS

// Server holds dependencies that handlers need.
type Server struct {
	store    *store.SQLiteStore
	pipeline *pipeline.Pipeline
	workdir  string
}

// NewServer creates a Server with its dependencies wired in.
func NewServer(s *store.SQLiteStore, p *pipeline.Pipeline, workdir string) *Server {
	return &Server{store: s, pipeline: p, workdir: workdir}
}

// respondJSON writes a status code and JSON body to the response.
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status) // must come before writing body
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("respondJSON encode failed: %v", err)
	}
}

// SetupRouter maps URL patterns to handler methods.
func (s *Server) SetupRouter() http.Handler {
	mux := http.NewServeMux()

	// serve the embedded HTML dashboard at root
	staticSub, _ := fs.Sub(staticFiles, "static")
	mux.Handle("GET /", http.FileServer(http.FS(staticSub)))

	mux.HandleFunc("GET /builds", s.ListBuilds)
	mux.HandleFunc("GET /builds/{id}", s.GetBuild)

	return mux
}

// ListBuilds — GET /builds
func (s *Server) ListBuilds(w http.ResponseWriter, _ *http.Request) {
	builds, err := s.store.List()
	if err != nil {
		http.Error(w, "could not list builds", http.StatusInternalServerError)
		return
	}
	respondJSON(w, http.StatusOK, builds)
}

// GetBuild — GET /builds/{id}
func (s *Server) GetBuild(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	build, err := s.store.Get(id)
	if err != nil {
		http.Error(w, "build not found", http.StatusNotFound)
		return
	}
	respondJSON(w, http.StatusOK, build)
}
