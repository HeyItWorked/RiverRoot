// Package store persists build results as JSON files on disk.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/HeyItWorked/riverroot/internal/pipeline"
	"github.com/google/uuid"
)

// BuildStore defines the interface for persisting build results.
type BuildStore interface {
	Save(result pipeline.BuildResult) (string, error)
	Get(id string) (pipeline.BuildResult, error)
}

// JSONStore implements BuildStore using JSON files on disk.
type JSONStore struct {
	dir string // directory where build files are stored
}

// NewJSONStore creates a JSONStore and ensures the storage directory exists.
func NewJSONStore(dir string) (*JSONStore, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &JSONStore{dir: dir}, nil
}

// Save marshals the build result to JSON and writes it to disk.
func (s *JSONStore) Save(result pipeline.BuildResult) (string, error) {
	uid := uuid.New().String()
	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	err = os.WriteFile(filepath.Join(s.dir, uid+".json"), data, 0644)
	if err != nil {
		return "", err
	}
	return uid, nil
}

// Get reads and unmarshals a single build result by ID.
func (s *JSONStore) Get(id string) (pipeline.BuildResult, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, id+".json"))
	if err != nil {
		return pipeline.BuildResult{}, err
	}
	var result pipeline.BuildResult
	if err := json.Unmarshal(data, &result); err != nil {
		return pipeline.BuildResult{}, err
	}
	return result, nil
}
