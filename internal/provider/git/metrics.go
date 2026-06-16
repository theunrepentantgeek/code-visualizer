package git

import (
	"path/filepath"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

const (
	FileAge           metric.Name = "file-age"
	FileFreshness     metric.Name = "file-freshness"
	AuthorCount       metric.Name = "author-count"
	CommitCount       metric.Name = "commit-count"
	TotalLinesAdded   metric.Name = "total-lines-added"
	TotalLinesRemoved metric.Name = "total-lines-removed"
	CommitDensity     metric.Name = "commit-density"

	// LinesAdded is a commit-level metric tracking lines added per commit.
	LinesAdded   metric.Name = "lines-added"
	LinesRemoved metric.Name = "lines-removed"
	LinesChanged metric.Name = "lines-changed"
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

// buildRelPathSet returns the set of relative paths (relative to the git
// worktree root) for all files under root.
func buildRelPathSet(s *repoService, root *model.Directory) map[string]bool {
	paths := make(map[string]bool)

	model.WalkFiles(root, func(f *model.File) {
		relPath, err := filepath.Rel(s.RepoRoot(), f.Path)
		if err == nil {
			paths[relPath] = true
		}
	})

	return paths
}
