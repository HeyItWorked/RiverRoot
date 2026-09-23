package runner

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

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
