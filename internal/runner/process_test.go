package runner

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
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

func TestRunCommand_Success(t *testing.T) {
	_, _, code, err := RunCommand("echo", []string{"hello"}, "", 5*time.Second)
	if err != nil {
		t.Fatalf("RunCommand(echo hello) err = %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestRunCommand_NonZeroExitIsNotAnError(t *testing.T) {
	_, _, code, err := RunCommand("false", nil, "", 5*time.Second)
	if err != nil {
		t.Fatalf("RunCommand(false) err = %v", err)
	}
	if code == 0 {
		t.Errorf("exit code = 0, want non-zero")
	}
}

func TestRunCommand_KillsCommandThatExceedsTimeout(t *testing.T) {
	const timeout = 200 * time.Millisecond
	start := time.Now()
	_, _, code, err := RunCommand("sleep", []string{"15"}, "", timeout)
	elapsed := time.Since(start)
	if elapsed > 5*time.Second {
		t.Errorf("took %v, want killed around %v", elapsed, timeout)
	}
	if err == nil && code == 0 {
		t.Errorf("reported success, want killed after %v", timeout)
	}
}

func TestRunCommand_RunsInGivenWorkdir(t *testing.T) {
	dir := t.TempDir()
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolving %q: %v", dir, err)
	}
	stdout, _, code, err := RunCommand("pwd", nil, dir, 5*time.Second)
	if err != nil {
		t.Fatalf("RunCommand(pwd) err = %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// stdout from bytes.Buffer.String() has trailing newline
	if got := stdout; got != want+"\n" {
		t.Errorf("stdout = %q, want %q", got, want+"\n")
	}
}

func TestRunCommand_EmptyWorkdirRunsInCurrentDirectory(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	want, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		t.Fatalf("resolving %q: %v", cwd, err)
	}
	stdout, _, code, err := RunCommand("pwd", nil, "", 5*time.Second)
	if err != nil {
		t.Fatalf("RunCommand(pwd) err = %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// stdout from bytes.Buffer.String() has trailing newline
	if got := stdout; got != want+"\n" {
		t.Errorf("stdout = %q, want %q", got, want+"\n")
	}
}

func TestRunStreaming_WritesOutputToWriter(t *testing.T) {
	tests := []struct {
		name      string
		cmd       string
		args      []string
		wantLines []string
	}{
		{"stdout", "echo", []string{"hello"}, []string{"hello"}},
		{"stderr", "sh", []string{"-c", "echo oops >&2"}, []string{"oops"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out syncBuffer
			code, err := RunStreaming(tt.cmd, tt.args, "", 5*time.Second, &out)
			if err != nil {
				t.Fatalf("RunStreaming(%s) err = %v", tt.cmd, err)
			}
			if code != 0 {
				t.Errorf("exit code = %d", code)
			}
			got := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
			if len(got) != len(tt.wantLines) {
				t.Fatalf("wrote %d lines, want %d", len(got), len(tt.wantLines))
			}
		})
	}
}

func TestRunStreaming_RunsInGivenWorkdir(t *testing.T) {
	dir := t.TempDir()
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolving %q: %v", dir, err)
	}
	var out syncBuffer
	code, err := RunStreaming("pwd", nil, dir, 5*time.Second, &out)
	if err != nil {
		t.Fatalf("RunStreaming(pwd) err = %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d", code)
	}
	// RunStreaming uses fmt.Fprintf with [timestamp] prefix and newline
	// Extract the path from the output (remove timestamp prefix)
	output := strings.TrimSpace(out.String())
	if i := strings.Index(output, "/"); i >= 0 {
		output = output[i:]
	}
	// Remove trailing newline from output for comparison
	output = strings.TrimSuffix(output, "\n")
	if output != want {
		t.Errorf("stdout = %q, want %q", output, want)
	}
}
