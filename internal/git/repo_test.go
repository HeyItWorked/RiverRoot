package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// newRepo makes a throwaway git repo with one commit per message.
// Each commit writes its own file so ChangedFiles has something to report.
func newRepo(t *testing.T, messages ...string) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=tester", "GIT_AUTHOR_EMAIL=t@example.com",
			"GIT_COMMITTER_NAME=tester", "GIT_COMMITTER_EMAIL=t@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	for i, msg := range messages {
		name := filepath.Join(dir, "file"+string(rune('a'+i))+".txt")
		if err := os.WriteFile(name, []byte(msg), 0o644); err != nil {
			t.Fatalf("writing %q: %v", name, err)
		}
		run("add", ".")
		run("commit", "-q", "-m", msg)
	}
	return dir
}

func headHash(t *testing.T, dir, rev string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "rev-parse", rev).Output()
	if err != nil {
		t.Fatalf("rev-parse %s: %v", rev, err)
	}
	return strings.TrimSpace(string(out))
}

func TestGetLatestCommit_ReturnsHeadCommit(t *testing.T) {
	dir := newRepo(t, "first commit", "second commit")
	hash, message, author, err := GetLatestCommit(dir)
	if err != nil {
		t.Fatalf("GetLatestCommit: %v", err)
	}
	if want := headHash(t, dir, "HEAD"); hash != want {
		t.Errorf("hash = %q, want %q", hash, want)
	}
	if message != "second commit" || author != "tester" {
		t.Errorf("message, author = %q, %q", message, author)
	}
}

func TestGetLatestCommit_NotARepoReturnsError(t *testing.T) {
	if _, _, _, err := GetLatestCommit(t.TempDir()); err == nil {
		t.Errorf("GetLatestCommit(empty dir) err = nil, want an error")
	}
}

func TestChangedFiles_ListsFilesBetweenCommits(t *testing.T) {
	dir := newRepo(t, "one", "two", "three")
	got, err := ChangedFiles(dir, headHash(t, dir, "HEAD~2"), headHash(t, dir, "HEAD"))
	if err != nil {
		t.Fatalf("ChangedFiles: %v", err)
	}
	if want := []string{"fileb.txt", "filec.txt"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ChangedFiles = %v, want %v", got, want)
	}
}
