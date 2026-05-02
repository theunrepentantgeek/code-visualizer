package git

import (
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
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
