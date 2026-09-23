// Package pipeline handles pipeline execution for riverroot.
// A pipeline runs steps in order.
package pipeline

import (
	"time"

	"github.com/HeyItWorked/riverroot/internal/runner"
)

// StepResult is what comes back from one pipeline step.
// Name, exit code, stdout/stderr, and Err (only if step couldn't run at all).
type StepResult struct {
	Name     string
	ExitCode int
	Stdout   string
	Stderr   string
	Err      error
}

// BuildResult is what comes back from a full pipeline run.
// ID and CreatedAt are only filled when loading from DB - Run() doesn't set these.
type BuildResult struct {
	ID        string
	CreatedAt string
	Pipeline  string
	Steps     []StepResult
	Failed    bool
}

// execStep runs one step as a local process in workdir.
// Non-zero exit is normal (Err=nil), only real failures come back as err.
// This was bug #31 — a bare `false` must fail the build.
func execStep(step Step, workdir string, timeout time.Duration) StepResult {
	stdout, stderr, code, err := runner.RunCommand("bash", []string{"-c", step.Command}, workdir, timeout)
	return StepResult{Name: step.Name, ExitCode: code, Stdout: stdout, Stderr: stderr, Err: err}
}

// stepFailed reports whether a finished step failed.
// This is a simple check: either Err is non-nil or ExitCode is non-zero.
func stepFailed(r StepResult) bool {
	return r.Err != nil || r.ExitCode != 0
}

// Run executes the pipeline's steps one at a time, in order.
// Run never returns an error; failures are recorded on BuildResult instead.
// Returns the final BuildResult with all step results and the failed flag.
func Run(p *Pipeline, workdir string, timeout time.Duration) BuildResult {
	var results []StepResult
	failed := false

	for _, step := range p.Steps {
		r := execStep(step, workdir, timeout)
		results = append(results, r)
		if stepFailed(r) {
			failed = true
			break
		}
	}

	return BuildResult{Pipeline: p.Name, Steps: results, Failed: failed}
}
