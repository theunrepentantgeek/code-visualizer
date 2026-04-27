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
	FileAge       metric.Name = "file-age"
	FileFreshness metric.Name = "file-freshness"
	AuthorCount   metric.Name = "author-count"
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
	case FileAge, FileFreshness, AuthorCount:
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

// loadGitMetric is the shared implementation for all git-based metric providers.
// It opens the repo service, walks all files, computes paths relative to the git
// worktree root (not the scan root), and sets the metric via the supplied fn.
func loadGitMetric(
	root *model.Directory,
	name metric.Name,
	desc string,
	fn func(*repoService, string) (int64, error),
	onFile func(),
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

		val, err := fn(s, relPath)
		if err != nil {
			if !errors.Is(err, errUntracked) {
				slog.Debug("could not get "+desc, "path", relPath, "error", err)
			}

			return
		}

		f.SetQuantity(name, val)
	})

	return nil
}
