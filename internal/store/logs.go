package store

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
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

// TailLog streams a build's log file to out, then watches for new lines (like tail -f).
// Stops when ctx is cancelled (e.g. client disconnects or build finishes).
func (l *LogStore) TailLog(ctx context.Context, buildID string, out io.Writer) error {
	path := filepath.Join(l.dir, buildID+".log")
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// catch up: read everything already in the file
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fmt.Fprintln(out, scanner.Text())
	}

	// poll: check for new lines every 200ms until cancelled
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(200 * time.Millisecond):
			for scanner.Scan() {
				fmt.Fprintln(out, scanner.Text())
			}
		}
	}
}
