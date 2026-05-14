package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
)

// validatePaths validates the target directory and output file paths.
func validatePaths(targetPath, output string) error {
	if _, err := canvas.FormatFromPath(output); err != nil {
		return &outputPathError{msg: err.Error()}
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &targetPathError{msg: "target path does not exist: " + targetPath}
		}

		return &targetPathError{msg: fmt.Sprintf("cannot access target path: %s", err)}
	}

	if !info.IsDir() {
		return &targetPathError{msg: "target path is not a directory: " + targetPath}
	}

	outDir := filepath.Dir(output)
	if outDir == "." {
		return nil
	}

	info, err = os.Stat(outDir)
	if err != nil {
		return &outputPathError{msg: "output directory does not exist: " + outDir}
	}

	if !info.IsDir() {
		return &outputPathError{msg: "output parent is not a directory: " + outDir}
	}

	return nil
}

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
		return &gitRequiredError{metric: metricLabel, target: targetPath}
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

// filterBinaryFiles removes binary files from the tree when the size metric
// is file-lines.
func filterBinaryFiles(sizeMetric string, root *model.Directory) error {
	if metric.Name(sizeMetric) != filesystem.FileLines {
		return nil
	}

	beforeCount, _ := countAll(root)
	filtered := scan.FilterBinaryFiles(root)
	afterCount, _ := countAll(filtered)
	excluded := beforeCount - afterCount
	slog.Debug("binary file filter", "excluded", excluded, "remaining", afterCount)

	if afterCount == 0 {
		return &noFilesAfterFilterError{
			msg: noFilesAfterFilterMsg,
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
