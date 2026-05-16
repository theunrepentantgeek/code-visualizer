package main

import (
	"log/slog"
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// buildFilterRules merges config-file filter rules with CLI --filter flags.
func buildFilterRules(cfg *config.Config, cliFilters []string) []filter.Rule {
	rules := make([]filter.Rule, 0, len(cfg.FileFilter)+len(cliFilters))
	rules = append(rules, cfg.FileFilter...)

	for _, f := range cliFilters {
		// Already validated in Validate()
		rule, _ := filter.ParseFilterFlag(f)
		rules = append(rules, rule)
	}

	return rules
}

// checkGitRequirement verifies a git repository exists when any requested
// metric needs git.
func checkGitRequirement(targetPath string, requested []metric.Name) error {
	name, needsGit := findGitMetric(requested)
	if !needsGit {
		return nil
	}

	return verifyGitRepo(targetPath, name)
}

// checkGitRepo verifies the target path is inside a git repository.
// Spiral always requires git for commit history.
func checkGitRepo(targetPath string) error {
	return verifyGitRepo(targetPath, "spiral")
}

func verifyGitRepo(targetPath string, metricLabel metric.Name) error {
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return eris.Wrap(err, "failed to resolve absolute path")
	}

	isGit, err := scan.IsGitRepo(absPath)
	if err != nil {
		return eris.Wrap(err, "git check failed")
	}

	if !isGit {
		return &stages.GitRequiredError{Metric: metricLabel, Target: targetPath}
	}

	return nil
}

// findGitMetric returns the first metric in the list that requires a git
// repository.
func findGitMetric(requested []metric.Name) (metric.Name, bool) {
	for _, name := range requested {
		if git.IsGitMetric(name) {
			return name, true
		}
	}

	return "", false
}

// filterBinaryFiles removes binary files from the tree unless includeBinary is true.
// Binary files are excluded by default because this is a code visualization tool;
// use --include-binary-files to include them.
func filterBinaryFiles(root *model.Directory) error {
	beforeCount, _ := countAll(root)
	filtered := scan.FilterBinaryFiles(root)
	afterCount, _ := countAll(filtered)
	excluded := beforeCount - afterCount
	slog.Debug("binary file filter", "excluded", excluded, "remaining", afterCount)

	if afterCount == 0 {
		return &stages.NoFilesAfterFilterError{
			Msg: stages.NoFilesAfterFilterMsg,
		}
	}

	// Update root in place — avoid struct copy which would copy the mutex.
	root.Files = filtered.Files
	root.Dirs = filtered.Dirs

	return nil
}

// resolveFillPalette determines the fill palette from config or provider
// defaults.
func resolveFillPalette(fill *config.MetricSpec, fillMetric metric.Name) palette.PaletteName {
	if fp := specPalette(fill); fp != "" {
		return fp
	}

	if d, ok := provider.GetDescriptor(fillMetric); ok {
		return d.DefaultPalette
	}

	return palette.Neutral
}

// resolveBorderMetricAndPalette determines the effective border metric and
// palette from config or provider defaults.
func resolveBorderMetricAndPalette(
	border *config.MetricSpec,
) (metric.Name, palette.PaletteName) {
	borderMetric := specMetric(border)
	if borderMetric == "" {
		return "", ""
	}

	borderPaletteName := specPalette(border)
	if borderPaletteName == "" {
		if d, ok := provider.GetDescriptor(borderMetric); ok {
			borderPaletteName = d.DefaultPalette
		} else {
			borderPaletteName = palette.Neutral
		}
	}

	return borderMetric, borderPaletteName
}
