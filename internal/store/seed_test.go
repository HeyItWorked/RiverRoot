package store

import "testing"

func TestSeedDemo_SavesSampleBuilds(t *testing.T) {
	s, err := NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	if err := SeedDemo(s); err != nil {
		t.Fatalf("SeedDemo: %v", err)
	}
	builds, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(builds) != 2 {
		t.Errorf("SeedDemo saved %d builds, want 2", len(builds))
	}
}
