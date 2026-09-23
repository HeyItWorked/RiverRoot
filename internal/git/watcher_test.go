package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestWatch_CallsOnChangeForNewCommit(t *testing.T) {
	dir := newRepo(t, "first commit")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	changed := make(chan string, 1)
	go Watch(ctx, dir, 20*time.Millisecond, func(hash string) { changed <- hash })

	// let Watch record the baseline before the new commit lands
	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0o644); err != nil {
		t.Fatalf("writing file: %v", err)
	}
	cmd := exec.Command("git", "-C", dir, "-c", "user.name=tester", "-c", "user.email=t@example.com",
		"commit", "-q", "--allow-empty", "-m", "second commit")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}

	select {
	case got := <-changed:
		if want := headHash(t, dir, "HEAD"); got != want {
			t.Errorf("onChange(%q), want %q", got, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("onChange was not called after a new commit")
	}
}

func TestWatch_ReturnsWhenContextCancelled(t *testing.T) {
	dir := newRepo(t, "first commit")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		Watch(ctx, dir, 20*time.Millisecond, func(string) {})
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Watch did not return after context cancelled")
	}
}
