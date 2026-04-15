package folder

import (
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/provider"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
)

// TotalFolderLinesProvider sums the line counts of all text files in a folder.
type TotalFolderLinesProvider struct{}

func (*TotalFolderLinesProvider) Name() metric.Name     { return TotalFolderLines }
func (*TotalFolderLinesProvider) Kind() metric.Kind     { return metric.Quantity }
func (*TotalFolderLinesProvider) Scope() provider.Scope { return provider.ScopeFolder }
func (*TotalFolderLinesProvider) Description() string {
	return "Total number of text lines in all contained files, including nested folders"
}

func (*TotalFolderLinesProvider) Dependencies() []metric.Name {
	return []metric.Name{filesystem.FileLines}
}
func (*TotalFolderLinesProvider) DefaultPalette() palette.PaletteName { return palette.Neutral }

func (*TotalFolderLinesProvider) Load(root *model.Directory) error {
	model.WalkDirectories(root, func(d *model.Directory) {
		total, found := sumQuantityFromFiles(d, filesystem.FileLines)
		if v, ok := sumQuantityFromDirs(d, TotalFolderLines); ok {
			total += v
			found = true
		}

		if found {
			d.SetQuantity(TotalFolderLines, total)
		}
	})

	return nil
}

// TotalFolderSizeProvider sums the file sizes of all files in a folder.
type TotalFolderSizeProvider struct{}

func (*TotalFolderSizeProvider) Name() metric.Name     { return TotalFolderSize }
func (*TotalFolderSizeProvider) Kind() metric.Kind     { return metric.Quantity }
func (*TotalFolderSizeProvider) Scope() provider.Scope { return provider.ScopeFolder }
func (*TotalFolderSizeProvider) Description() string {
	return "Total Size in bytes of all contained files, including nested folders"
}

func (*TotalFolderSizeProvider) Dependencies() []metric.Name {
	return []metric.Name{filesystem.FileSize}
}
func (*TotalFolderSizeProvider) DefaultPalette() palette.PaletteName { return palette.Neutral }

func (*TotalFolderSizeProvider) Load(root *model.Directory) error {
	model.WalkDirectories(root, func(d *model.Directory) {
		total, found := sumQuantityFromFiles(d, filesystem.FileSize)
		if v, ok := sumQuantityFromDirs(d, TotalFolderSize); ok {
			total += v
			found = true
		}

		if found {
			d.SetQuantity(TotalFolderSize, total)
		}
	})

	return nil
}

// MeanFileLinesProvider reports the mean line count of text files in a folder, skipping binary files.
type MeanFileLinesProvider struct{}

func (*MeanFileLinesProvider) Name() metric.Name     { return MeanFileLines }
func (*MeanFileLinesProvider) Kind() metric.Kind     { return metric.Measure }
func (*MeanFileLinesProvider) Scope() provider.Scope { return provider.ScopeFolder }
func (*MeanFileLinesProvider) Description() string {
	return "Mean count of file lines in all contained text files, including nested folders"
}

func (*MeanFileLinesProvider) Dependencies() []metric.Name {
	return []metric.Name{filesystem.FileLines}
}
func (*MeanFileLinesProvider) DefaultPalette() palette.PaletteName { return palette.Neutral }

func (*MeanFileLinesProvider) Load(root *model.Directory) error {
	stats := make(map[string]*sumCount)

	model.WalkDirectories(root, func(d *model.Directory) {
		sc := &sumCount{}

		addPositiveSumCountFromFiles(sc, d, filesystem.FileLines)
		addSumCountFromDirs(sc, d, stats)

		stats[d.Path] = sc

		if sc.count > 0 {
			d.SetMeasure(MeanFileLines, sc.sum/float64(sc.count))
		}
	})

	return nil
}

// MeanFileSizeProvider reports the mean file size in bytes of all files in a folder.
type MeanFileSizeProvider struct{}

func (*MeanFileSizeProvider) Name() metric.Name     { return MeanFileSize }
func (*MeanFileSizeProvider) Kind() metric.Kind     { return metric.Measure }
func (*MeanFileSizeProvider) Scope() provider.Scope { return provider.ScopeFolder }
func (*MeanFileSizeProvider) Description() string {
	return "Mean size in bytes of all files, including nested folders"
}
func (*MeanFileSizeProvider) Dependencies() []metric.Name         { return []metric.Name{filesystem.FileSize} }
func (*MeanFileSizeProvider) DefaultPalette() palette.PaletteName { return palette.Neutral }

func (*MeanFileSizeProvider) Load(root *model.Directory) error {
	stats := make(map[string]*sumCount)

	model.WalkDirectories(root, func(d *model.Directory) {
		sc := &sumCount{}

		addSumCountFromFiles(sc, d, filesystem.FileSize)
		addSumCountFromDirs(sc, d, stats)

		stats[d.Path] = sc

		if sc.count > 0 {
			d.SetMeasure(MeanFileSize, sc.sum/float64(sc.count))
		}
	})

	return nil
}
