package git

import (
	"errors"
	"log/slog"
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
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

// FileAgeProvider reports time since first commit in days.
type FileAgeProvider struct {
	onFile func()
}

func (*FileAgeProvider) Name() metric.Name { return FileAge }
func (*FileAgeProvider) Kind() metric.Kind { return metric.Quantity }
func (*FileAgeProvider) Description() string {
	return "Time since first commit (days); older files score higher."
}
func (*FileAgeProvider) Dependencies() []metric.Name         { return nil }
func (*FileAgeProvider) DefaultPalette() palette.PaletteName { return palette.Temperature }

func (p *FileAgeProvider) SetOnFileProcessed(fn func()) { p.onFile = fn }

func (p *FileAgeProvider) Load(root *model.Directory) error {
	return loadGitMetric(root, FileAge, "file-age", (*repoService).fileAge, p.onFile)
}

// FileFreshnessProvider reports time since most recent commit in days.
type FileFreshnessProvider struct {
	onFile func()
}

func (*FileFreshnessProvider) Name() metric.Name { return FileFreshness }
func (*FileFreshnessProvider) Kind() metric.Kind { return metric.Quantity }
func (*FileFreshnessProvider) Description() string {
	return "Time since most recent commit (days); recently changed files score higher."
}
func (*FileFreshnessProvider) Dependencies() []metric.Name         { return nil }
func (*FileFreshnessProvider) DefaultPalette() palette.PaletteName { return palette.Temperature }

func (p *FileFreshnessProvider) SetOnFileProcessed(fn func()) { p.onFile = fn }

func (p *FileFreshnessProvider) Load(root *model.Directory) error {
	return loadGitMetric(root, FileFreshness, "file-freshness", (*repoService).fileFreshness, p.onFile)
}

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

// AuthorCountProvider reports the number of distinct commit authors.
type AuthorCountProvider struct {
	onFile func()
}

func (*AuthorCountProvider) Name() metric.Name { return AuthorCount }
func (*AuthorCountProvider) Kind() metric.Kind { return metric.Quantity }
func (*AuthorCountProvider) Description() string {
	return "Number of distinct commit authors; files touched by many people score higher."
}
func (*AuthorCountProvider) Dependencies() []metric.Name         { return nil }
func (*AuthorCountProvider) DefaultPalette() palette.PaletteName { return palette.GoodBad }

func (p *AuthorCountProvider) SetOnFileProcessed(fn func()) { p.onFile = fn }

func (p *AuthorCountProvider) Load(root *model.Directory) error {
	return loadGitMetric(root, AuthorCount, "author-count", (*repoService).authorCount, p.onFile)
}

// CommitCountProvider reports the total number of commits that modified each file.
type CommitCountProvider struct {
	onFile func()
}

func (*CommitCountProvider) Name() metric.Name { return CommitCount }
func (*CommitCountProvider) Kind() metric.Kind { return metric.Quantity }
func (*CommitCountProvider) Description() string {
	return "Number of commits that modified the file; frequently changed files score higher."
}
func (*CommitCountProvider) Dependencies() []metric.Name         { return nil }
func (*CommitCountProvider) DefaultPalette() palette.PaletteName { return palette.Temperature }

func (p *CommitCountProvider) SetOnFileProcessed(fn func()) { p.onFile = fn }

func (p *CommitCountProvider) Load(root *model.Directory) error {
	return loadGitMetric(root, CommitCount, "commit-count", (*repoService).commitCount, p.onFile)
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
