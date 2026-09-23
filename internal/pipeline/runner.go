// Package pipeline handles pipeline execution for riverroot.
// A pipeline runs steps in order, with parallel steps running together.
package pipeline

import (
	"fmt"
	"sync"
	"time"

	"github.com/HeyItWorked/riverroot/internal/runner"
	"golang.org/x/sync/errgroup"
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

// runBatch runs steps in a batch concurrently.
// Returns true if any step in the batch failed.
// Uses errgroup to manage the goroutines and a mutex to protect the results slice.
func runBatch(batch []Step, workdir string, timeout time.Duration, results *[]StepResult) bool {
	g := errgroup.Group{}
	var mu sync.Mutex // protects results slice from concurrent writes

	for _, s := range batch {
		step := s // capture loop var
		g.Go(func() error {
			r := execStep(step, workdir, timeout)

			mu.Lock()
			*results = append(*results, r)
			mu.Unlock()

			if r.Err != nil {
				return r.Err
			}
			if r.ExitCode != 0 {
				return fmt.Errorf("step %q exited %d", step.Name, r.ExitCode)
			}
			return nil
		})
	}

	return g.Wait() != nil
}

// Run executes the pipeline's steps.
// Sequential steps run one at a time, parallel steps run as a batch.
// Run never returns an error; failures are recorded on BuildResult instead.
// Returns the final BuildResult with all step results and the failed flag.
func Run(p *Pipeline, workdir string, timeout time.Duration) BuildResult {
	var results []StepResult
	failed := false

	i := 0
	for i < len(p.Steps) {
		if !p.Steps[i].Parallel {
			// sequential step — run and wait before moving on
			r := execStep(p.Steps[i], workdir, timeout)
			results = append(results, r)
			if stepFailed(r) {
				failed = true
				break
			}
			i++
			continue
		}

		// collect consecutive parallel steps into a batch
		var batch []Step
		for i < len(p.Steps) && p.Steps[i].Parallel {
			batch = append(batch, p.Steps[i])
			i++
		}

		if runBatch(batch, workdir, timeout, &results) {
			failed = true
			break
		}
	}

	return BuildResult{Pipeline: p.Name, Steps: results, Failed: failed}
}
