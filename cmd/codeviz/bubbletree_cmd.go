package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/bubbletree"
	"github.com/bevan/code-visualizer/internal/canvas"
	"github.com/bevan/code-visualizer/internal/config"
	"github.com/bevan/code-visualizer/internal/export"
	"github.com/bevan/code-visualizer/internal/filter"
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/provider"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
	"github.com/bevan/code-visualizer/internal/scan"
)

type BubbletreeCmd struct {
	TargetPath string `arg:"" help:"Path to directory to scan."`
	Output     string `help:"Output image file path (png, jpg, jpeg, svg)." required:"true" short:"o"`

	//nolint:revive // kong struct tags require long lines
	Size metric.Name `default:"" help:"Metric for circle size; run 'codeviz help-metrics' for available metrics." short:"s"`

	Fill config.MetricSpec `help:"Fill colour: metric[,palette] (e.g. file-type,categorization)." optional:"" short:"f"`

	Border config.MetricSpec `help:"Border colour: metric[,palette] (e.g. file-lines,foliage)." optional:"" short:"b"`

	Labels string `enum:",all,folders,none" default:"" help:"Labels to display: all, folders, or none."`

	//nolint:revive // kong struct tags require long lines
	Legend string `default:"" enum:",top-left,top-center,top-right,center-right,bottom-right,bottom-center,bottom-left,center-left,none" help:"Legend position (default: bottom-right)." optional:""`

	//nolint:revive // kong struct tags require long lines
	LegendOrientation string `default:"" enum:",vertical,horizontal" help:"Legend orientation (auto-detected from position if omitted)." name:"legend-orientation" optional:""`

	Width  int `default:"1920" help:"Image width in pixels."`
	Height int `default:"1080" help:"Image height in pixels."`

	Filter []string `help:"Filter rule: glob to include, !glob to exclude (repeatable, order-preserved)."` //nolint:revive // kong struct tags require long lines
}

func (c *BubbletreeCmd) Validate() error {
	for _, f := range c.Filter {
		if _, err := filter.ParseFilterFlag(f); err != nil {
			return eris.Wrapf(err, "invalid filter %q", f)
		}
	}

	return nil
}

// validateConfig checks the effective configuration after all sources have been
// merged. Called from mergeConfigAndValidate() after TryAutoLoad + applyOverrides.
func (*BubbletreeCmd) validateConfig(cfg *config.Bubbletree) error {
	size := ptrString(cfg.Size)

	p, ok := provider.Get(metric.Name(size))
	if !ok {
		return eris.Errorf("unknown size metric %q; available metrics: %s", size, formatMetricNames())
	}

	if p.Kind() != metric.Quantity && p.Kind() != metric.Measure {
		return eris.Errorf("size metric must be numeric, got %q (kind: %d)", size, p.Kind())
	}

	if err := cfg.Fill.Validate("fill"); err != nil {
		return eris.Wrap(err, "invalid fill spec")
	}

	if err := cfg.Border.Validate("border"); err != nil {
		return eris.Wrap(err, "invalid border spec")
	}

	return nil
}

// mergeConfigAndValidate loads the config file, merges CLI overrides on top,
// and validates the effective configuration. Called at the start of Run().
func (c *BubbletreeCmd) mergeConfigAndValidate(flags *Flags) error {
	if err := flags.Config.TryAutoLoad(c.Output); err != nil {
		return eris.Wrap(err, "auto-config load failed")
	}

	c.applyOverrides(flags.Config)

	return c.validateConfig(flags.Config.Bubbletree)
}

//nolint:dupl // parallel Run methods on different config types share the same workflow
func (c *BubbletreeCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	cfg := flags.Config.Bubbletree
	size := metric.Name(ptrString(cfg.Size))

	if err := c.validatePaths(); err != nil {
		return err
	}

	if flags.ExportConfig != "" {
		if err := flags.Config.Save(flags.ExportConfig); err != nil {
			return eris.Wrap(err, "failed to save config")
		}
	}

	fillMetric := c.resolveFillMetric(cfg)
	fillPaletteName := c.resolveFillPalette(cfg, fillMetric)

	filterRules := c.buildFilterRules(flags.Config)

	slog.Info("Scanning filesystem", "path", c.TargetPath)

	scanProg, stopScanTicker := buildScanProgress(flags)

	root, err := scan.Scan(c.TargetPath, filterRules, scanProg)

	stopScanTicker()

	if err != nil {
		return eris.Wrap(err, "scan failed")
	}

	requested := collectRequestedMetrics(size, cfg.Fill, cfg.Border)

	if err := c.checkGitRequirement(requested); err != nil {
		return err
	}

	slog.Info("Calculating metrics")

	metricProg, stopMetricTicker := buildMetricProgress(flags, model.CountFiles(root))

	if err := provider.Run(root, requested, metricProg); err != nil {
		stopMetricTicker()

		return eris.Wrap(err, "failed to load metrics")
	}

	stopMetricTicker()

	if err := c.filterBinaryFiles(cfg, root); err != nil {
		return err
	}

	if err := export.Export(root, requested, flags.ExportData); err != nil {
		return eris.Wrap(err, "failed to export data")
	}

	width := ptrInt(flags.Config.Width, 1920)
	height := ptrInt(flags.Config.Height, 1080)

	return c.renderAndLog(root, cfg, width, height, fillMetric, fillPaletteName)
}

func (c *BubbletreeCmd) renderAndLog(
	root *model.Directory,
	cfg *config.Bubbletree,
	width, height int,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
) error {
	size := metric.Name(ptrString(cfg.Size))
	files, dirs := countAll(root)

	slog.Info("Rendering image", "output", c.Output, "width", width, "height", height)

	borderMetric, borderPaletteName := c.resolveBorderMetricAndPalette(cfg)

	labels := c.resolveLabels(cfg)
	nodes := bubbletree.Layout(root, width, height, size, labels)
	inks := buildBubbleInks(root, fillMetric, fillPaletteName, borderMetric, borderPaletteName)
	cv := renderBubbleToCanvas(&nodes, root, width, height, inks)

	if err := cv.Render(c.Output); err != nil {
		return eris.Wrap(err, "render failed")
	}

	slog.Info("Rendered bubble tree",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"width", width,
		"height", height,
		"size_metric", string(size),
		"fill_metric", string(fillMetric),
		"fill_palette", string(fillPaletteName),
		"border_metric", string(borderMetric),
		"border_palette", string(borderPaletteName),
	)

	return nil
}

// applyOverrides writes non-zero CLI flag values on top of the config layer.
// Zero-valued CLI fields are transparent — the config value passes through unchanged.
func (c *BubbletreeCmd) applyOverrides(cfg *config.Config) {
	cfg.OverrideWidth(c.Width)
	cfg.OverrideHeight(c.Height)

	if cfg.Bubbletree == nil {
		cfg.Bubbletree = &config.Bubbletree{}
	}

	cfg.Bubbletree.OverrideSize(string(c.Size))
	cfg.Bubbletree.OverrideFill(c.Fill)
	cfg.Bubbletree.OverrideBorder(c.Border)
	cfg.Bubbletree.OverrideLabels(c.Labels)
	cfg.Bubbletree.OverrideLegend(c.Legend)
	cfg.Bubbletree.OverrideLegendOrientation(c.LegendOrientation)
}

//nolint:dupl // mirrors TreemapCmd.validatePaths by design
func (c *BubbletreeCmd) validatePaths() error {
	if _, err := canvas.FormatFromPath(c.Output); err != nil {
		return &outputPathError{msg: err.Error()}
	}

	info, err := os.Stat(c.TargetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &targetPathError{msg: "target path does not exist: " + c.TargetPath}
		}

		return &targetPathError{msg: fmt.Sprintf("cannot access target path: %s", err)}
	}

	if !info.IsDir() {
		return &targetPathError{msg: "target path is not a directory: " + c.TargetPath}
	}

	outDir := filepath.Dir(c.Output)
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

func (c *BubbletreeCmd) buildFilterRules(cfg *config.Config) []filter.Rule {
	rules := make([]filter.Rule, 0, len(cfg.FileFilter)+len(c.Filter))
	rules = append(rules, cfg.FileFilter...)

	for _, f := range c.Filter {
		// Already validated in Validate()
		rule, _ := filter.ParseFilterFlag(f)
		rules = append(rules, rule)
	}

	return rules
}

func (c *BubbletreeCmd) checkGitRequirement(requested []metric.Name) error {
	name, needsGit := findGitMetric(requested)
	if !needsGit {
		return nil
	}

	absPath, err := filepath.Abs(c.TargetPath)
	if err != nil {
		return eris.Wrap(err, "failed to resolve absolute path")
	}

	isGit, err := scan.IsGitRepo(absPath)
	if err != nil {
		return eris.Wrap(err, "git check failed")
	}

	if !isGit {
		return &gitRequiredError{metric: name, target: c.TargetPath}
	}

	return nil
}

func (*BubbletreeCmd) resolveFillMetric(cfg *config.Bubbletree) metric.Name {
	if fill := specMetric(cfg.Fill); fill != "" {
		return fill
	}

	return metric.Name(ptrString(cfg.Size))
}

func (*BubbletreeCmd) resolveFillPalette(cfg *config.Bubbletree, fillMetric metric.Name) palette.PaletteName {
	if fp := specPalette(cfg.Fill); fp != "" {
		return fp
	}

	if p, ok := provider.Get(fillMetric); ok {
		return p.DefaultPalette()
	}

	return palette.Neutral
}

func (*BubbletreeCmd) resolveBorderMetricAndPalette(
	cfg *config.Bubbletree,
) (metric.Name, palette.PaletteName) {
	border := specMetric(cfg.Border)
	if border == "" {
		return "", ""
	}

	borderPaletteName := specPalette(cfg.Border)
	if borderPaletteName == "" {
		if p, ok := provider.Get(border); ok {
			borderPaletteName = p.DefaultPalette()
		} else {
			borderPaletteName = palette.Neutral
		}
	}

	return border, borderPaletteName
}

func (*BubbletreeCmd) filterBinaryFiles(cfg *config.Bubbletree, root *model.Directory) error {
	if metric.Name(ptrString(cfg.Size)) != filesystem.FileLines {
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

// resolveLabels converts the string labels flag to a bubbletree.LabelMode.
func (*BubbletreeCmd) resolveLabels(cfg *config.Bubbletree) bubbletree.LabelMode {
	if lbl := ptrString(cfg.Labels); lbl != "" {
		return bubbletree.LabelMode(lbl)
	}

	return bubbletree.LabelFoldersOnly
}
