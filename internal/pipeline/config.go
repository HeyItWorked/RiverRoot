// Package pipeline parses pipeline.yaml and runs the steps it declares —
// sequentially or in parallel batches — collecting a result for each step.
package pipeline

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Step is one command to run in the pipeline.
type Step struct {
	Name     string `yaml:"name"`
	Command  string `yaml:"command"`
	Parallel bool   `yaml:"parallel"`
	Image    string `yaml:"image"`
}

// Pipeline is the parsed representation of pipeline.yaml.
// *Pipeline can be nil on failure — a value type (Pipeline) always exists as a zero value and can never be nil.
// Returning a pointer lets us express "nothing to give you" cleanly alongside error.
type Pipeline struct {
	Name  string `yaml:"name"`
	Steps []Step `yaml:"steps"`
}

// Load reads a pipeline.yaml file at path and returns the parsed Pipeline.
func Load(path string) (*Pipeline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// zero-value Pipeline — fields will be filled by Unmarshal
	// unmarshal = parse raw bytes into a struct (opposite of marshal: struct → bytes)
	var p Pipeline
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	return &p, nil
}
