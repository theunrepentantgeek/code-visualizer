package git

import (
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
)

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
