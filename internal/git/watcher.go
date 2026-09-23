package git

import (
	"context"
	"time"
)

// Watch polls repoPath every interval and calls onChange when the commit hash changes.
// Runs until ctx is cancelled.
func Watch(ctx context.Context, repoPath string, interval time.Duration, onChange func(hash string)) {
	lastHash, _, _, err := GetLatestCommit(repoPath)
	baselineSet := err == nil

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			newHash, _, _, err := GetLatestCommit(repoPath)
			if err != nil {
				continue // intentionally skip bad ticks — transient git errors shouldn't crash the watcher
			}
			if !baselineSet {
				lastHash = newHash
				baselineSet = true
				continue
			}

			if newHash != lastHash {
				onChange(newHash)
				lastHash = newHash
			}
		}
	}
}
