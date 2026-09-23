package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// LogStore manages log files for builds.
// It writes logs to individual files named after the build ID.
type LogStore struct {
	dir string
}

// NewLogStore creates a LogStore and ensures the log directory exists.
// Uses os.MkdirAll so it creates parent directories as needed.
func NewLogStore(dir string) (*LogStore, error) {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}
	return &LogStore{dir: dir}, nil
}

// WriteLog appends a single log line to <dir>/<buildID>.log
func (l *LogStore) WriteLog(buildID string, line string) error {
	path := filepath.Join(l.dir, buildID+".log")

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintln(file, line)
	return err
}
