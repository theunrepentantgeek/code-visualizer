package git

import (
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
)

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
