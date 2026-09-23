// Package git inspects a repository by shelling out to the git command:
// reading the latest commit, listing files changed between two commits,
// and polling the head hash for changes.
package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetLatestCommit returns the most recent commit's hash, message, and author
// for the git repo at repoPath.
func GetLatestCommit(repoPath string) (hash, message, author string, err error) {
	// -C runs git as if it were in repoPath without changing working directory
	// %H=full hash, %s=subject, %an=author name, %n=newline separator
	cmd := exec.Command("git", "-C", repoPath, "log", "-1", "--format=%H%n%s%n%an")
	out, execErr := cmd.Output()
	if execErr != nil {
		err = fmt.Errorf("git log failed: %w", execErr)
		return
	}

	// SplitN(..., 3) stops at 3 parts so a multi-line commit message doesn't break the parse
	trimmed := strings.TrimSpace(string(out))
	parts := strings.SplitN(trimmed, "\n", 3)
	hash, message, author = parts[0], parts[1], parts[2]

	return
}

// ChangedFiles returns the list of files changed between two commits.
func ChangedFiles(repoPath, fromHash, toHash string) ([]string, error) {
	// fromHash..toHash is git's range syntax — all changes introduced between the two commits
	cmd := exec.Command("git", "-C", repoPath, "diff", "--name-only", fromHash+".."+toHash)
	out, execErr := cmd.Output()
	if execErr != nil {
		return nil, fmt.Errorf("git diff failed: %w", execErr)
	}

	// each filename is on its own line — Split gives us a slice directly
	trimmed := strings.TrimSpace(string(out))
	parts := strings.Split(trimmed, "\n")

	return parts, nil
}
