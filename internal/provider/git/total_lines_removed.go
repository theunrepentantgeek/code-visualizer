package git

import (
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
)

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
