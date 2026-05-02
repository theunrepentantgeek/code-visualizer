package git

import (
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
)

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
