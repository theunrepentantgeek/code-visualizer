package folder

import (
	"log/slog"
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/provider"
	gitprovider "github.com/bevan/code-visualizer/internal/provider/git"
)

// FolderAuthorCountProvider reports the count of distinct authors across all files in a folder.
type FolderAuthorCountProvider struct{}

func (*FolderAuthorCountProvider) Name() metric.Name     { return FolderAuthorCount }
func (*FolderAuthorCountProvider) Kind() metric.Kind     { return metric.Quantity }
func (*FolderAuthorCountProvider) Scope() provider.Scope { return provider.ScopeFolder }
func (*FolderAuthorCountProvider) Description() string {
	return "Count of distinct authors who have contributed to folder"
}

func (*FolderAuthorCountProvider) Dependencies() []metric.Name {
	return []metric.Name{gitprovider.AuthorCount}
}
func (*FolderAuthorCountProvider) DefaultPalette() palette.PaletteName { return palette.GoodBad }

func (*FolderAuthorCountProvider) Load(root *model.Directory) error {
	if err := gitprovider.VerifyRepository(root.Path); err != nil {
		return eris.Wrap(err, "folder-author-count requires a git repository")
	}

	fileAuthors := collectFileAuthors(root)
	applyFolderAuthorCounts(root, fileAuthors)

	return nil
}

// collectFileAuthors walks all files and returns a map of file path → set of author emails.
func collectFileAuthors(root *model.Directory) map[string]map[string]bool {
	fileAuthors := make(map[string]map[string]bool)

	model.WalkFiles(root, func(f *model.File) {
		relPath, err := filepath.Rel(root.Path, f.Path)
		if err != nil {
			slog.Warn("could not compute relative path", "path", f.Path, "error", err)

			return
		}

		authors, err := gitprovider.FileAuthors(root.Path, relPath)
		if err != nil {
			slog.Debug("could not get file authors", "path", relPath, "error", err)

			return
		}

		if len(authors) > 0 {
			fileAuthors[f.Path] = authors
		}
	})

	return fileAuthors
}

// applyFolderAuthorCounts sets folder-author-count on every directory using bottom-up union.
func applyFolderAuthorCounts(root *model.Directory, fileAuthors map[string]map[string]bool) {
	dirAuthors := make(map[string]map[string]bool)

	model.WalkDirectories(root, func(d *model.Directory) {
		all := unionAuthors(d, fileAuthors, dirAuthors)
		dirAuthors[d.Path] = all

		if len(all) > 0 {
			d.SetQuantity(FolderAuthorCount, int64(len(all)))
		}
	})
}

// unionAuthors builds the union of author sets from direct files and subdirs of d.
func unionAuthors(
	d *model.Directory,
	fileAuthors map[string]map[string]bool,
	dirAuthors map[string]map[string]bool,
) map[string]bool {
	all := make(map[string]bool)

	for _, f := range d.Files {
		for author := range fileAuthors[f.Path] {
			all[author] = true
		}
	}

	for _, sub := range d.Dirs {
		for author := range dirAuthors[sub.Path] {
			all[author] = true
		}
	}

	return all
}

// FolderAgeProvider reports days since the oldest file in the folder was first committed.
type FolderAgeProvider struct{}

func (*FolderAgeProvider) Name() metric.Name     { return FolderAge }
func (*FolderAgeProvider) Kind() metric.Kind     { return metric.Quantity }
func (*FolderAgeProvider) Scope() provider.Scope { return provider.ScopeFolder }
func (*FolderAgeProvider) Description() string {
	return "Number of days since the folder was first committed to git"
}
func (*FolderAgeProvider) Dependencies() []metric.Name         { return []metric.Name{gitprovider.FileAge} }
func (*FolderAgeProvider) DefaultPalette() palette.PaletteName { return palette.Temperature }

func (*FolderAgeProvider) Load(root *model.Directory) error {
	loadMaxQuantity(root, gitprovider.FileAge, FolderAge)

	return nil
}

// FolderFreshnessProvider reports days since the most recently modified file in the folder.
type FolderFreshnessProvider struct{}

func (*FolderFreshnessProvider) Name() metric.Name     { return FolderFreshness }
func (*FolderFreshnessProvider) Kind() metric.Kind     { return metric.Quantity }
func (*FolderFreshnessProvider) Scope() provider.Scope { return provider.ScopeFolder }
func (*FolderFreshnessProvider) Description() string {
	return "Number of days since the folder was last modified in git"
}

func (*FolderFreshnessProvider) Dependencies() []metric.Name {
	return []metric.Name{gitprovider.FileFreshness}
}
func (*FolderFreshnessProvider) DefaultPalette() palette.PaletteName { return palette.Temperature }

func (*FolderFreshnessProvider) Load(root *model.Directory) error {
	loadMinQuantity(root, gitprovider.FileFreshness, FolderFreshness)

	return nil
}

// MeanFileAgeProvider reports the mean file age (days) across all files in a folder.
type MeanFileAgeProvider struct{}

func (*MeanFileAgeProvider) Name() metric.Name     { return MeanFileAge }
func (*MeanFileAgeProvider) Kind() metric.Kind     { return metric.Measure }
func (*MeanFileAgeProvider) Scope() provider.Scope { return provider.ScopeFolder }
func (*MeanFileAgeProvider) Description() string {
	return "Mean age of all contained files, including nested folders"
}
func (*MeanFileAgeProvider) Dependencies() []metric.Name         { return []metric.Name{gitprovider.FileAge} }
func (*MeanFileAgeProvider) DefaultPalette() palette.PaletteName { return palette.Temperature }

func (*MeanFileAgeProvider) Load(root *model.Directory) error {
	loadMeanMeasure(root, gitprovider.FileAge, MeanFileAge)

	return nil
}

// MeanFileFreshnessProvider reports the mean file freshness (days) across all files in a folder.
type MeanFileFreshnessProvider struct{}

func (*MeanFileFreshnessProvider) Name() metric.Name     { return MeanFileFreshness }
func (*MeanFileFreshnessProvider) Kind() metric.Kind     { return metric.Measure }
func (*MeanFileFreshnessProvider) Scope() provider.Scope { return provider.ScopeFolder }
func (*MeanFileFreshnessProvider) Description() string {
	return "Mean freshness of all contained files, including nested folders"
}

func (*MeanFileFreshnessProvider) Dependencies() []metric.Name {
	return []metric.Name{gitprovider.FileFreshness}
}
func (*MeanFileFreshnessProvider) DefaultPalette() palette.PaletteName { return palette.Temperature }

func (*MeanFileFreshnessProvider) Load(root *model.Directory) error {
	loadMeanMeasure(root, gitprovider.FileFreshness, MeanFileFreshness)

	return nil
}
