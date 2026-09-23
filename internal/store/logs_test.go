package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func waitFor(t *testing.T, out *syncBuffer, want string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if out.String() == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("log output = %q, want %q", out.String(), want)
}

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

func TestLogStore_WriteThenTailReturnsContent(t *testing.T) {
	l, err := NewLogStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLogStore: %v", err)
	}
	const buildID = "build-1"
	lines := []string{"step one started", "step one finished"}
	for _, line := range lines {
		if err := l.WriteLog(buildID, line); err != nil {
			t.Fatalf("WriteLog(%q): %v", line, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var out syncBuffer
	done := make(chan error, 1)
	go func() { done <- l.TailLog(ctx, buildID, &out) }()
	waitFor(t, &out, strings.Join(lines, "\n")+"\n")
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("TailLog returned %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("TailLog did not return after context cancelled")
	}
}

func TestLogStore_TailLogReturnsOnContextCancel(t *testing.T) {
	l, err := NewLogStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLogStore: %v", err)
	}
	const buildID = "build-1"
	if err := l.WriteLog(buildID, "still running"); err != nil {
		t.Fatalf("WriteLog: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var out syncBuffer
	done := make(chan error, 1)
	go func() { done <- l.TailLog(ctx, buildID, &out) }()
	waitFor(t, &out, "still running\n")
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("TailLog returned %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("TailLog did not return within 2s")
	}
}

func TestLogStore_TailLogUnknownBuildReturnsError(t *testing.T) {
	l, err := NewLogStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLogStore: %v", err)
	}
	var out syncBuffer
	if err := l.TailLog(context.Background(), "no-such-build", &out); err == nil {
		t.Errorf("TailLog(unknown build) err = nil, want an error")
	}
	if got := out.String(); got != "" {
		t.Errorf("TailLog(unknown build) wrote %q", got)
	}
}
