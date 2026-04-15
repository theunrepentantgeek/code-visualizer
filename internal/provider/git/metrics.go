package git

import (
	"errors"
	"log/slog"
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/provider"
)

const (
	FileAge       metric.Name = "file-age"
	FileFreshness metric.Name = "file-freshness"
	AuthorCount   metric.Name = "author-count"
)

// FileAgeProvider reports the number of days since the file's first commit.
type FileAgeProvider struct{}

func (*FileAgeProvider) Name() metric.Name     { return FileAge }
func (*FileAgeProvider) Kind() metric.Kind     { return metric.Quantity }
func (*FileAgeProvider) Scope() provider.Scope { return provider.ScopeFile }
func (*FileAgeProvider) Description() string {
	return "Number of days since the file was first committed to git"
}
func (*FileAgeProvider) Dependencies() []metric.Name         { return nil }
func (*FileAgeProvider) DefaultPalette() palette.PaletteName { return palette.Temperature }

func (*FileAgeProvider) Load(root *model.Directory) error {
	s, err := getService(root.Path)
	if err != nil {
		return eris.Wrap(err, "file-age requires a git repository")
	}

	model.WalkFiles(root, func(f *model.File) {
		relPath, err := filepath.Rel(root.Path, f.Path)
		if err != nil {
			slog.Warn("could not compute relative path", "path", f.Path, "error", err)

			return
		}

		age, err := s.fileAge(relPath)
		if err != nil {
			if !errors.Is(err, errUntracked) {
				slog.Debug("could not get file age", "path", relPath, "error", err)
			}

			return
		}

		f.SetQuantity(FileAge, age)
	})

	return nil
}

// FileFreshnessProvider reports the number of days since the file's most recent commit.
type FileFreshnessProvider struct{}

func (*FileFreshnessProvider) Name() metric.Name     { return FileFreshness }
func (*FileFreshnessProvider) Kind() metric.Kind     { return metric.Quantity }
func (*FileFreshnessProvider) Scope() provider.Scope { return provider.ScopeFile }
func (*FileFreshnessProvider) Description() string {
	return "Number of days since the file was last modified in git"
}
func (*FileFreshnessProvider) Dependencies() []metric.Name         { return nil }
func (*FileFreshnessProvider) DefaultPalette() palette.PaletteName { return palette.Temperature }

func (*FileFreshnessProvider) Load(root *model.Directory) error {
	s, err := getService(root.Path)
	if err != nil {
		return eris.Wrap(err, "file-freshness requires a git repository")
	}

	model.WalkFiles(root, func(f *model.File) {
		relPath, err := filepath.Rel(root.Path, f.Path)
		if err != nil {
			slog.Warn("could not compute relative path", "path", f.Path, "error", err)

			return
		}

		freshness, err := s.fileFreshness(relPath)
		if err != nil {
			if !errors.Is(err, errUntracked) {
				slog.Debug("could not get file freshness", "path", relPath, "error", err)
			}

			return
		}

		f.SetQuantity(FileFreshness, freshness)
	})

	return nil
}

// AuthorCountProvider reports the number of distinct commit authors for a file.
type AuthorCountProvider struct{}

func (*AuthorCountProvider) Name() metric.Name     { return AuthorCount }
func (*AuthorCountProvider) Kind() metric.Kind     { return metric.Quantity }
func (*AuthorCountProvider) Scope() provider.Scope { return provider.ScopeFile }
func (*AuthorCountProvider) Description() string {
	return "Count of distinct authors who have contributed to this file"
}
func (*AuthorCountProvider) Dependencies() []metric.Name         { return nil }
func (*AuthorCountProvider) DefaultPalette() palette.PaletteName { return palette.GoodBad }

func (*AuthorCountProvider) Load(root *model.Directory) error {
	s, err := getService(root.Path)
	if err != nil {
		return eris.Wrap(err, "author-count requires a git repository")
	}

	model.WalkFiles(root, func(f *model.File) {
		relPath, err := filepath.Rel(root.Path, f.Path)
		if err != nil {
			slog.Warn("could not compute relative path", "path", f.Path, "error", err)

			return
		}

		count, err := s.authorCount(relPath)
		if err != nil {
			if !errors.Is(err, errUntracked) {
				slog.Debug("could not get author count", "path", relPath, "error", err)
			}

			return
		}

		f.SetQuantity(AuthorCount, count)
	})

	return nil
}
