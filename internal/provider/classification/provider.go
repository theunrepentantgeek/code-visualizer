// Package classification provides metric providers that classify files by
// matching their relative path against user-defined glob patterns.
//
// Providers in this package are created from [config.SelectionMetric] entries
// rather than registered via init(); call [NewProvider] for each entry in
// [config.Config.SelectionMetricsList] and pass the result to
// [provider.Register].
package classification

import (
	"path/filepath"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// Provider classifies each file by matching its relative path against an ordered
// list of glob patterns.  The category string of the first matching rule is
// stored as a Classification metric value.  Files that match no rule receive
// no value for this metric.
type Provider struct {
	name  metric.Name
	rules []config.SelectionMetricRule
}

// NewProvider creates a Provider from a [config.SelectionMetric].
// The returned provider should be passed to [provider.Register] before
// the pipeline runs.
func NewProvider(cfg config.SelectionMetric) *Provider {
	return &Provider{
		name:  metric.Name(cfg.Name),
		rules: cfg.Rules,
	}
}

func (p *Provider) Name() metric.Name                   { return p.name }
func (p *Provider) Kind() metric.Kind                   { return metric.Classification }
func (p *Provider) Target() metric.Target               { return metric.File }
func (p *Provider) Description() string                 { return "User-defined filename-based file classification." }
func (p *Provider) Dependencies() []metric.Name         { return nil }
func (p *Provider) DefaultPalette() palette.PaletteName { return palette.Categorization }

// Load walks every file in root and sets the classification metric for files
// that match at least one rule.
func (p *Provider) Load(root *model.Directory) error {
	model.WalkFiles(root, func(f *model.File) {
		relPath, err := filepath.Rel(root.Path, f.Path)
		if err != nil {
			relPath = f.Path
		}

		// Normalize to forward slashes for consistent matching across platforms.
		relPath = filepath.ToSlash(relPath)

		for _, rule := range p.rules {
			matched, err := filter.MatchPattern(rule.Filename, relPath)
			if err != nil {
				// Invalid pattern — skip silently; config validation should
				// catch malformed patterns before we reach this point.
				continue
			}

			if matched {
				f.SetClassification(p.name, rule.Category)

				return
			}
		}
	})

	return nil
}
