package git

import (
	"log/slog"
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/walk"
)

const (
	FileAge           metric.Name = "file-age"
	FileFreshness     metric.Name = "file-freshness"
	AuthorCount       metric.Name = "author-count"
	CommitCount       metric.Name = "commit-count"
	TotalLinesAdded   metric.Name = "total-lines-added"
	TotalLinesRemoved metric.Name = "total-lines-removed"
	CommitDensity     metric.Name = "commit-density"
)

// IsGitMetric reports whether name is a metric that requires a git repository.
func IsGitMetric(name metric.Name) bool {
	switch name {
	case FileAge, FileFreshness, AuthorCount, CommitCount,
		TotalLinesAdded, TotalLinesRemoved, CommitDensity:
		return true
	default:
		return false
	}
}

// walkGitFiles opens the repo service, walks all files, computes paths relative
// to the git worktree root, and invokes the process callback for each file.
// It bulk-prewarms the commit cache before iterating so that all getCommitData
// calls hit the cache rather than issuing individual git-log queries per file.
func walkGitFiles(
	root *model.Directory,
	desc string,
	onFile func(),
	process func(*repoService, *model.File, string),
) error {
	s, err := getService(root.Path)
	if err != nil {
		return eris.Wrapf(err, "%s requires a git repository", desc)
	}

	// Build the set of relative paths for all tracked files and prewarm the
	// commit cache in a single git log pass.
	pathSet := buildRelPathSet(s, root)
	if prewarmErr := s.bulkPrewarm(pathSet); prewarmErr != nil {
		// Prewarm failure is non-fatal: per-file fallback still works.
		slog.Warn("git bulk prewarm failed, falling back to per-file lookups",
			"metric", desc, "error", prewarmErr)
	}

	walk.Files(root, func(f *model.File) {
		if onFile != nil {
			defer onFile()
		}

		relPath, err := filepath.Rel(s.RepoRoot(), f.Path)
		if err != nil {
			slog.Warn("could not compute relative path", "path", f.Path, "error", err)

			return
		}

		process(s, f, relPath)
	})

	return nil
}

// buildRelPathSet returns the set of relative paths (relative to the git
// worktree root) for all files under root.
func buildRelPathSet(s *repoService, root *model.Directory) map[string]bool {
	paths := make(map[string]bool)

	walk.Files(root, func(f *model.File) {
		relPath, err := filepath.Rel(s.RepoRoot(), f.Path)
		if err == nil {
			paths[relPath] = true
		}
	})

	return paths
}
