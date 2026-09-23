package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLogStore_WriteLogAppendsLines(t *testing.T) {
	dir := t.TempDir()
	l, err := NewLogStore(dir)
	if err != nil {
		t.Fatalf("NewLogStore: %v", err)
	}
	for _, line := range []string{"step one started", "step one finished"} {
		if err := l.WriteLog("build-1", line); err != nil {
			t.Fatalf("WriteLog(%q): %v", line, err)
		}
	}
	got, err := os.ReadFile(filepath.Join(dir, "build-1.log"))
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	if want := "step one started\nstep one finished\n"; string(got) != want {
		t.Errorf("log file = %q, want %q", got, want)
	}
}
