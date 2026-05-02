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
