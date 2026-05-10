package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/canvas"
	"github.com/bevan/code-visualizer/internal/config"
	"github.com/bevan/code-visualizer/internal/export"
	"github.com/bevan/code-visualizer/internal/filter"
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/provider"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
	"github.com/bevan/code-visualizer/internal/radialtree"
	"github.com/bevan/code-visualizer/internal/scan"
)

type RadialCmd struct {
	TargetPath string `arg:"" help:"Path to directory to scan."`
	Output     string `help:"Output image file path (png, jpg, jpeg, svg)." required:"true" short:"o"`

	DiscSize metric.Name `default:"" help:"Metric for disc size; run 'codeviz help-metrics' for available metrics." short:"d"` //nolint:revive,nolintlint // kong struct tags require long lines

	Fill   config.MetricSpec `help:"Fill colour: metric[,palette] (e.g. file-type,categorization)." optional:"" short:"f"` //nolint:revive,nolintlint // kong struct tags require long lines
	Border config.MetricSpec `help:"Border colour: metric[,palette] (e.g. file-lines,foliage)." optional:"" short:"b"`     //nolint:revive,nolintlint // kong struct tags require long lines

	Labels string `enum:",all,folders,none" default:"" help:"Labels to display: all, folders, or none."`

	Legend            string `default:"" enum:",top-left,top-center,top-right,center-right,bottom-right,bottom-center,bottom-left,center-left,none" help:"Legend position (default: bottom-right)." optional:""` //nolint:revive // kong struct tags require long lines
	LegendOrientation string `default:"" enum:",vertical,horizontal" help:"Legend orientation (auto-detected from position if omitted)." name:"legend-orientation" optional:""`                                  //nolint:revive // kong struct tags require long lines

	Width  int `default:"1920" help:"Image width in pixels."`
	Height int `default:"1920" help:"Image height in pixels."`

	Filter []string `help:"Filter rule: glob to include, !glob to exclude (repeatable, order-preserved)."` //nolint:revive // kong struct tags require long lines
}

func (c *RadialCmd) Validate() error {
	for _, f := range c.Filter {
		if _, err := filter.ParseFilterFlag(f); err != nil {
			return eris.Wrapf(err, "invalid filter %q", f)
		}
	}

	return nil
}

// validateConfig checks the effective configuration after all sources have been
// merged. Called from mergeConfigAndValidate() after TryAutoLoad + applyOverrides.
func (*RadialCmd) validateConfig(cfg *config.Radial) error {
	discSize := ptrString(cfg.DiscSize)

	p, ok := provider.Get(metric.Name(discSize))
	if !ok {
		return eris.Errorf("unknown disc-size metric %q; available metrics: %s", discSize, formatMetricNames())
	}

	if p.Kind() != metric.Quantity && p.Kind() != metric.Measure {
		return eris.Errorf("disc-size metric must be numeric, got %q (kind: %d)", discSize, p.Kind())
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
func (c *RadialCmd) mergeConfigAndValidate(flags *Flags) error {
	if err := flags.Config.TryAutoLoad(c.Output); err != nil {
		return eris.Wrap(err, "auto-config load failed")
	}

	c.applyOverrides(flags.Config)

	return c.validateConfig(flags.Config.Radial)
}

func (c *RadialCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	cfg := flags.Config.Radial
	discSize := metric.Name(ptrString(cfg.DiscSize))

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

	requested := collectRequestedMetrics(discSize, cfg.Fill, cfg.Border)

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

	files, dirs := countAll(root)

	canvasSize := min(ptrInt(flags.Config.Width, 1920), ptrInt(flags.Config.Height, 1920))

	return c.renderAndLog(root, cfg, files, dirs, canvasSize, fillMetric, fillPaletteName)
}

func (c *RadialCmd) renderAndLog(
	root *model.Directory,
	cfg *config.Radial,
	files, dirs, canvasSize int,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
) error {
	discSize := metric.Name(ptrString(cfg.DiscSize))
	labels := c.resolveLabels(cfg)
	nodes := radialtree.Layout(root, canvasSize, discSize, labels)

	borderMetric, borderPaletteName := c.resolveBorderMetricAndPalette(cfg)

	inks := buildRadialInks(root, fillMetric, fillPaletteName, borderMetric, borderPaletteName)

	slog.Info("Rendering image", "output", c.Output, "canvas_size", canvasSize)

	cv := renderRadialToCanvas(&nodes, root, canvasSize, inks)

	legendPos, legendOrient := resolveLegendOptions(ptrString(cfg.Legend), ptrString(cfg.LegendOrientation))
	legendConfig := buildLegendConfig(
		legendPos, legendOrient,
		inks.fill, fillMetric,
		inks.border, borderMetric,
		metric.Name(ptrString(cfg.DiscSize)),
	)

	if legendConfig != nil {
		cv.SetLegend(*legendConfig)
	}

	if err := cv.Render(c.Output); err != nil {
		return eris.Wrap(err, "render failed")
	}

	slog.Info("Rendered radial tree",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"canvas_size", canvasSize,
		"disc_metric", string(discSize),
		"fill_metric", string(fillMetric),
		"fill_palette", string(fillPaletteName),
		"border_metric", string(borderMetric),
		"border_palette", string(borderPaletteName),
	)

	return nil
}

func (*RadialCmd) resolveBorderMetricAndPalette(cfg *config.Radial) (metric.Name, palette.PaletteName) {
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

// applyOverrides writes non-zero CLI flag values on top of the config layer.
// Zero-valued CLI fields are transparent — the config value passes through unchanged.
func (c *RadialCmd) applyOverrides(cfg *config.Config) {
	cfg.OverrideWidth(c.Width)
	cfg.OverrideHeight(c.Height)

	if cfg.Radial == nil {
		cfg.Radial = &config.Radial{}
	}

	cfg.Radial.OverrideDiscSize(string(c.DiscSize))
	cfg.Radial.OverrideFill(c.Fill)
	cfg.Radial.OverrideBorder(c.Border)
	cfg.Radial.OverrideLabels(c.Labels)
	cfg.Radial.OverrideLegend(c.Legend)
	cfg.Radial.OverrideLegendOrientation(c.LegendOrientation)
}

//nolint:dupl // mirrors TreemapCmd.validatePaths by design
func (c *RadialCmd) validatePaths() error {
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

func (c *RadialCmd) buildFilterRules(cfg *config.Config) []filter.Rule {
	rules := make([]filter.Rule, 0, len(cfg.FileFilter)+len(c.Filter))
	rules = append(rules, cfg.FileFilter...)

	for _, f := range c.Filter {
		// Already validated in Validate()
		rule, _ := filter.ParseFilterFlag(f)
		rules = append(rules, rule)
	}

	return rules
}

func (c *RadialCmd) checkGitRequirement(requested []metric.Name) error {
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

func (*RadialCmd) resolveFillMetric(cfg *config.Radial) metric.Name {
	if fill := specMetric(cfg.Fill); fill != "" {
		return fill
	}

	return metric.Name(ptrString(cfg.DiscSize))
}

func (*RadialCmd) resolveFillPalette(cfg *config.Radial, fillMetric metric.Name) palette.PaletteName {
	if fp := specPalette(cfg.Fill); fp != "" {
		return fp
	}

	if p, ok := provider.Get(fillMetric); ok {
		return p.DefaultPalette()
	}

	return palette.Neutral
}

func (*RadialCmd) filterBinaryFiles(cfg *config.Radial, root *model.Directory) error {
	if metric.Name(ptrString(cfg.DiscSize)) != filesystem.FileLines {
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

// resolveLabels converts the string labels flag to a radialtree.LabelMode.
func (*RadialCmd) resolveLabels(cfg *config.Radial) radialtree.LabelMode {
	if lbl := ptrString(cfg.Labels); lbl != "" {
		return radialtree.LabelMode(lbl)
	}

	return radialtree.LabelAll
}
