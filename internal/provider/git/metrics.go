package git

import (
	"errors"
	"log/slog"
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
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

// loadGitMetric is the shared implementation for all git-based quantity metric providers.
func loadGitMetric(
	root *model.Directory,
	name metric.Name,
	desc string,
	fn func(*repoService, string) (int64, error),
	onFile func(),
) error {
	return walkGitFiles(root, desc, onFile, func(s *repoService, f *model.File, relPath string) {
		val, err := fn(s, relPath)
		if err != nil {
			if !errors.Is(err, errUntracked) {
				slog.Debug("could not get "+desc, "path", relPath, "error", err)
			}

			return
		}

		f.SetQuantity(name, val)
	})
}

// loadGitMeasureMetric is the shared implementation for git-based measure (float64) providers.
func loadGitMeasureMetric(
	root *model.Directory,
	name metric.Name,
	desc string,
	fn func(*repoService, string) (float64, error),
	onFile func(),
) error {
	return walkGitFiles(root, desc, onFile, func(s *repoService, f *model.File, relPath string) {
		val, err := fn(s, relPath)
		if err != nil {
			if !errors.Is(err, errUntracked) {
				slog.Debug("could not get "+desc, "path", relPath, "error", err)
			}

			return
		}

		f.SetMeasure(name, val)
	})
}

// walkGitFiles opens the repo service, walks all files, computes paths relative
// to the git worktree root, and invokes the process callback for each file.
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

	model.WalkFiles(root, func(f *model.File) {
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
