package pipeline

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		fileBody string
		skipFile bool
		wantErr  bool
		want     *Pipeline
	}{
		{
			name:     "valid file fills every field",
			fileBody: "name: my-app\nsteps:\n  - name: test\n    command: go test ./...\n    image: golang:1.22\n    parallel: true\n  - name: build\n    command: go build ./...\n",
			want: &Pipeline{
				Name: "my-app",
				Steps: []Step{
					{Name: "test", Command: "go test ./...", Parallel: true, Image: "golang:1.22"},
					{Name: "build", Command: "go build ./..."},
				},
			},
		},
		{
			name:     "malformed yaml returns an error",
			fileBody: "name: my-app\nsteps: [unclosed\n",
			wantErr:  true,
		},
		{
			name:     "missing file returns an error",
			skipFile: true,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "pipeline.yaml")
			if !tt.skipFile {
				if err := os.WriteFile(path, []byte(tt.fileBody), 0o644); err != nil {
					t.Fatalf("writing fixture: %v", err)
				}
			}
			got, err := Load(path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Load(%q) err = nil", path)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load(%q) unexpected error: %v", path, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Load(%q)", path)
			}
		})
	}
}
