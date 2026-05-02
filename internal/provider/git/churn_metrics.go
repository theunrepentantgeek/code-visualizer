package git

import (
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
)

// TotalLinesAddedProvider reports accumulated lines added over all commits
// (excluding the first commit which creates the file).
type TotalLinesAddedProvider struct {
	onFile func()
}

func (*TotalLinesAddedProvider) Name() metric.Name { return TotalLinesAdded }
func (*TotalLinesAddedProvider) Kind() metric.Kind { return metric.Quantity }
func (*TotalLinesAddedProvider) Description() string {
	return "Accumulated lines added over all commits (excluding file creation); high churn files score higher."
}
func (*TotalLinesAddedProvider) Dependencies() []metric.Name         { return nil }
func (*TotalLinesAddedProvider) DefaultPalette() palette.PaletteName { return palette.Temperature }

func (p *TotalLinesAddedProvider) SetOnFileProcessed(fn func()) { p.onFile = fn }

func (p *TotalLinesAddedProvider) Load(root *model.Directory) error {
	return loadGitMetric(root, TotalLinesAdded, "total-lines-added", (*repoService).totalLinesAdded, p.onFile)
}

// TotalLinesRemovedProvider reports accumulated lines removed over all commits.
type TotalLinesRemovedProvider struct {
	onFile func()
}

func (*TotalLinesRemovedProvider) Name() metric.Name { return TotalLinesRemoved }
func (*TotalLinesRemovedProvider) Kind() metric.Kind { return metric.Quantity }
func (*TotalLinesRemovedProvider) Description() string {
	return "Accumulated lines removed over all commits; high churn files score higher."
}
func (*TotalLinesRemovedProvider) Dependencies() []metric.Name         { return nil }
func (*TotalLinesRemovedProvider) DefaultPalette() palette.PaletteName { return palette.Temperature }

func (p *TotalLinesRemovedProvider) SetOnFileProcessed(fn func()) { p.onFile = fn }

func (p *TotalLinesRemovedProvider) Load(root *model.Directory) error {
	return loadGitMetric(root, TotalLinesRemoved, "total-lines-removed", (*repoService).totalLinesRemoved, p.onFile)
}

// CommitDensityProvider reports commits per month of file lifetime.
type CommitDensityProvider struct {
	onFile func()
}

func (*CommitDensityProvider) Name() metric.Name { return CommitDensity }
func (*CommitDensityProvider) Kind() metric.Kind { return metric.Measure }
func (*CommitDensityProvider) Description() string {
	return "Commits per month of file lifetime; frequently changed files score higher."
}
func (*CommitDensityProvider) Dependencies() []metric.Name         { return nil }
func (*CommitDensityProvider) DefaultPalette() palette.PaletteName { return palette.Temperature }

func (p *CommitDensityProvider) SetOnFileProcessed(fn func()) { p.onFile = fn }

func (p *CommitDensityProvider) Load(root *model.Directory) error {
	return loadGitMeasureMetric(root, CommitDensity, "commit-density", (*repoService).commitDensity, p.onFile)
}
