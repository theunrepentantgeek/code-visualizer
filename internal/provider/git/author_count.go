package git

import (
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
)

// AuthorCountProvider reports the number of distinct commit authors.
type AuthorCountProvider struct {
	onFile func()
}

func (*AuthorCountProvider) Name() metric.Name { return AuthorCount }
func (*AuthorCountProvider) Kind() metric.Kind { return metric.Quantity }
func (*AuthorCountProvider) Description() string {
	return "Number of distinct commit authors; files touched by many people score higher."
}
func (*AuthorCountProvider) Dependencies() []metric.Name         { return nil }
func (*AuthorCountProvider) DefaultPalette() palette.PaletteName { return palette.GoodBad }

func (p *AuthorCountProvider) SetOnFileProcessed(fn func()) { p.onFile = fn }

func (p *AuthorCountProvider) Load(root *model.Directory) error {
	return loadGitMetric(root, AuthorCount, "author-count", (*repoService).authorCount, p.onFile)
}
