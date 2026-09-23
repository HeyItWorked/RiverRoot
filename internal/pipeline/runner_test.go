package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRun_StepsRunInListedOrder checks steps run in order.
func TestRun_StepsRunInListedOrder(t *testing.T) {
	dir := t.TempDir()
	orderFile := filepath.Join(dir, "order.txt")
	p := &Pipeline{
		Name: "order",
		Steps: []Step{
			{Name: "one", Command: fmt.Sprintf("echo one >> %q", orderFile)},
			{Name: "two", Command: fmt.Sprintf("echo two >> %q", orderFile)},
			{Name: "three", Command: fmt.Sprintf("echo three >> %q", orderFile)},
		},
	}
	res := Run(p, dir, 10*time.Second)
	if res.Failed {
		t.Errorf("Run(...) Failed = true, want false")
	}
	if len(res.Steps) != len(p.Steps) {
		t.Fatalf("Run(...) returned %d step results, want %d", len(res.Steps), len(p.Steps))
	}
	got, err := os.ReadFile(orderFile)
	if err != nil {
		t.Fatalf("reading order file: %v", err)
	}
	if want := "one\ntwo\nthree\n"; string(got) != want {
		t.Errorf("steps executed in order %q, want %q", got, want)
	}
}

// TestRun_FailedStepStopsRun checks a failing step stops the run.
func TestRun_FailedStepStopsRun(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		wantCode int
	}{
		{"false fails", "false", 1},
		{"exit with stderr", "echo boom >&2; exit 7", 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			marker := filepath.Join(dir, "never-ran.txt")
			p := &Pipeline{
				Name: "failing",
				Steps: []Step{
					{Name: "warmup", Command: "true"},
					{Name: "boom", Command: tt.command},
					{Name: "never", Command: fmt.Sprintf("echo reached >> %q", marker)},
				},
			}
			res := Run(p, dir, 10*time.Second)
			if !res.Failed {
				t.Errorf("Run(...) Failed = false, want true")
			}
			if len(res.Steps) != 2 {
				t.Fatalf("Run(...) returned %d step results, want 2", len(res.Steps))
			}
			if s := res.Steps[0]; s.Name != "warmup" || s.ExitCode != 0 {
				t.Errorf("first step = %+v", s)
			}
			s := res.Steps[1]
			if s.Name != "boom" || s.ExitCode != tt.wantCode {
				t.Errorf("failed step = %+v", s)
			}
			if _, err := os.Stat(marker); err == nil {
				t.Errorf("step after failed one ran (%q exists)", marker)
			}
		})
	}
}

// TestRun_RecordsStepResults checks the per-step results.
func TestRun_RecordsStepResults(t *testing.T) {
	table := []struct {
		name       string
		command    string
		wantStdout string
		wantStderr string
	}{
		{"greet", "echo hi", "hi\n", ""},
		{"complain", "echo oops 1>&2", "", "oops\n"},
		{"quiet", "true", "", ""},
	}
	p := &Pipeline{Name: "results"}
	for _, tt := range table {
		p.Steps = append(p.Steps, Step{Name: tt.name, Command: tt.command})
	}
	res := Run(p, t.TempDir(), 10*time.Second)
	if res.Failed {
		t.Errorf("Run(...) Failed = true, want false")
	}
	if len(res.Steps) != len(table) {
		t.Fatalf("Run(...) returned %d step results, want %d", len(res.Steps), len(table))
	}
	for i, tt := range table {
		s := res.Steps[i]
		if s.Name != tt.name || s.ExitCode != 0 || s.Stdout != tt.wantStdout || s.Stderr != tt.wantStderr {
			t.Errorf("step %d = %+v", i, s)
		}
	}
}
