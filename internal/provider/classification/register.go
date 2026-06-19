// Package classification provides metric loaders for user-defined,
// filename-glob-based file classification metrics.
//
// Unlike static providers, selection metrics are defined in the user's config
// file at runtime. Call [Register] after the config is loaded to register base
// metric descriptors and loaders for all selection metrics defined in the config.
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

// Register registers base metric descriptors and loaders for all selection
// metrics defined in cfg. Metrics that are already registered are skipped,
// making this safe to call multiple times with the same config.
//
// This must be called after the config is fully loaded and before
// provider.RunLoaders is called.
func Register(cfg *config.Config) {
	for _, sm := range cfg.SelectionMetricsList() {
		name := metric.Name(sm.Name)

		// Skip metrics that were already registered (idempotent).
		if _, exists := provider.GetBase(name); exists {
			continue
		}

		rules := sm.Rules // capture for closure

		provider.RegisterBase(provider.BaseMetricDescriptor{
			Name:           name,
			Kind:           metric.Classification,
			Level:          metric.LevelFile,
			Description:    "User-defined filename-based file classification.",
			Aggregations:   []metric.AggregationName{metric.AggMode, metric.AggDistinct},
			DefaultPalette: palette.Categorization,
		})

		provider.RegisterLoader(provider.BaseMetricLoader{
			Metrics: []metric.Name{name},
			Load:    makeLoadFunc(name, rules),
		})
	}
}

func makeLoadFunc(name metric.Name, rules []config.SelectionMetricRule) provider.LoadFunc {
	return func(root *model.Directory) error {
		model.WalkFiles(root, func(f *model.File) {
			if cat, ok := matchFirstRule(root.Path, f.Path, rules); ok {
				f.SetClassification(name, cat)
			}
		})

		return nil
	}
}

// matchFirstRule returns the category of the first rule whose pattern matches
// filePath relative to rootPath, or ("", false) if no rule matches.
func matchFirstRule(rootPath, filePath string, rules []config.SelectionMetricRule) (string, bool) {
	relPath, err := filepath.Rel(rootPath, filePath)
	if err != nil {
		relPath = filePath
	}

	// Normalize to forward slashes for consistent cross-platform matching.
	relPath = filepath.ToSlash(relPath)

	for _, rule := range rules {
		matched, matchErr := filter.MatchPattern(rule.Filename, relPath)
		if matchErr != nil {
			// Invalid pattern — skip silently; config validation should
			// catch malformed patterns before we reach this point.
			continue
		}

		if matched {
			return rule.Category, true
		}
	}

	return "", false
}
