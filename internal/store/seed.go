package store

import (
	"github.com/HeyItWorked/riverroot/internal/pipeline"
)

// SeedDemo adds some initial builds for demo purposes
// These are only added when DEMO=1 is set
func SeedDemo(s BuildStore) error {
	// some sample builds to populate the dashboard
	builds := []pipeline.BuildResult{
		{
			Pipeline: "demo-app",
			Steps: []pipeline.StepResult{
				{Name: "build", ExitCode: 0, Stdout: "go build: success\n"},
				{Name: "test", ExitCode: 1, Stdout: "FAIL: some test\n"},
			},
			Failed: true,
		},
		{
			Pipeline: "another-app",
			Steps: []pipeline.StepResult{
				{Name: "build", ExitCode: 0, Stdout: "go build: success\n"},
				{Name: "test", ExitCode: 0, Stdout: "ok: all tests passed\n"},
			},
			Failed: false,
		},
	}

	for _, b := range builds {
		if _, err := s.Save(b); err != nil {
			return err
		}
	}
	return nil
}
