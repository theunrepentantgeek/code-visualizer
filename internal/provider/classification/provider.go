// Package classification provides metric providers that classify files by
// matching their relative path against user-defined glob patterns.
//
// Providers in this package are created from [config.SelectionMetric] entries
// rather than registered via init(); call [Register] for each entry in
// [config.Config.SelectionMetricsList] to register both the descriptor and
// loader for that metric.
package classification

import (
	"path/filepath"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// loader classifies each file by matching its relative path against an ordered
// list of glob patterns.  The category string of the first matching rule is
// stored as a Classification metric value.  Files that match no rule receive
// no value for this metric.
type loader struct {
	name  metric.Name
	rules []config.SelectionMetricRule
}

// Register registers a base metric descriptor and loader for the given
// [config.SelectionMetric]. Call once per entry in
// [config.Config.SelectionMetricsList].
func Register(cfg config.SelectionMetric) {
	name := metric.Name(cfg.Name)

	provider.RegisterBase(provider.BaseMetricDescriptor{
		Name:           name,
		Kind:           metric.Classification,
		Level:          metric.LevelFile,
		Description:    "User-defined filename-based file classification.",
		DefaultPalette: palette.Categorization,
	})

	l := &loader{name: name, rules: cfg.Rules}

	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{name},
		Load:    l.Load,
	})
}

// Load walks every file in root and sets the classification metric for files
// that match at least one rule.
func (l *loader) Load(root *model.Directory) error {
	model.WalkFiles(root, func(f *model.File) {
		relPath, err := filepath.Rel(root.Path, f.Path)
		if err != nil {
			relPath = f.Path
		}

		// Normalize to forward slashes for consistent matching across platforms.
		relPath = filepath.ToSlash(relPath)

		for _, rule := range l.rules {
			matched, err := filter.MatchPattern(rule.Filename, relPath)
			if err != nil {
				// Invalid pattern — skip silently; config validation should
				// catch malformed patterns before we reach this point.
				continue
			}

			if matched {
				f.SetClassification(l.name, rule.Category)

				return
			}
		}
	})

	return nil
}
