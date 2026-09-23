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

// Run executes the pipeline's steps one at a time, in order.
// Returns the final BuildResult with all step results.
func Run(p *Pipeline, workdir string, timeout time.Duration) BuildResult {
	var results []StepResult

	for _, step := range p.Steps {
		results = append(results, execStep(step, workdir, timeout))
	}

	return BuildResult{Pipeline: p.Name, Steps: results}
}
