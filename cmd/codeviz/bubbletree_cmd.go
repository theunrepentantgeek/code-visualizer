package main

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/bubbletree"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/export"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
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

	Filter             []string `help:"Filter rule: glob to include, !glob to exclude (repeatable, order-preserved)."`                            //nolint:revive // kong struct tags require long lines
	IncludeBinaryFiles bool     `help:"Include binary files in the visualization (excluded by default)." name:"include-binary-files" optional:""` //nolint:revive // kong struct tags require long lines
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

	d, ok := provider.GetDescriptor(metric.Name(size))
	if !ok {
		return eris.Errorf("unknown size metric %q; available metrics: %s", size, formatMetricNames())
	}

	if d.Kind != metric.Quantity && d.Kind != metric.Measure {
		return eris.Errorf("size metric must be numeric, got %q (kind: %d)", size, d.Kind)
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

//nolint:revive,cyclop,funlen // Run methods share workflow structure across visualization commands
func (c *BubbletreeCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	cfg := flags.Config.Bubbletree
	size := metric.Name(ptrString(cfg.Size))

	if err := stages.ValidatePathsHelper(c.TargetPath, c.Output); err != nil {
		return eris.Wrap(err, "path validation failed")
	}

	if flags.ExportConfig != "" {
		if err := flags.Config.Save(flags.ExportConfig); err != nil {
			return eris.Wrap(err, "failed to save config")
		}
	}

	fillMetric := c.resolveFillMetric(cfg)
	fillPaletteName := stages.ResolveFillPalette(cfg.Fill, fillMetric)

	filterRules := stages.BuildFilterRulesHelper(flags.Config, c.Filter)

	slog.Info("Scanning filesystem", "path", c.TargetPath)

	scanProg, stopScanTicker := stages.BuildScanProgress(toStagesFlags(flags))

	root, err := scan.Scan(c.TargetPath, filterRules, scanProg)

	stopScanTicker()

	if err != nil {
		return eris.Wrap(err, "scan failed")
	}

	requested := stages.CollectRequestedMetrics(size, cfg.Fill, cfg.Border)

	if err := stages.CheckGitRequirementHelper(c.TargetPath, requested); err != nil {
		return eris.Wrap(err, "git requirement check failed")
	}

	slog.Info("Calculating metrics")

	metricProg, stopMetricTicker := stages.BuildMetricProgress(toStagesFlags(flags), model.CountFiles(root))

	if err := provider.Run(root, requested, metricProg); err != nil {
		stopMetricTicker()

		return eris.Wrap(err, "failed to load metrics")
	}

	stopMetricTicker()

	if !c.IncludeBinaryFiles {
		if err := stages.FilterBinaryFilesHelper(root); err != nil {
			return eris.Wrap(err, "binary file filter failed")
		}
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
	files, dirs := stages.CountAll(root)

	slog.Info("Rendering image", "output", c.Output, "width", width, "height", height)

	borderMetric, borderPaletteName := stages.ResolveBorderMetricAndPalette(cfg.Border)

	labels := c.resolveLabels(cfg)
	nodes := bubbletree.Layout(root, width, height, size, labels)
	inks := bubbletree.BuildInks(root, fillMetric, fillPaletteName, borderMetric, borderPaletteName)
	cv := bubbletree.RenderToCanvas(&nodes, root, width, height, inks)

	legendPos, legendOrient := legend.ResolveOptions(ptrString(cfg.Legend), ptrString(cfg.LegendOrientation))
	legendConfig := legend.Build(
		legendPos, legendOrient,
		inks.Fill, fillMetric,
		inks.Border, borderMetric,
		size,
	)

	if legendConfig != nil {
		cv.SetLegend(*legendConfig)
	}

	if err := cv.Render(c.Output); err != nil {
		return eris.Wrap(err, "render failed")
	}

	slog.Info(
		"Rendered bubble tree",
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

func (*BubbletreeCmd) resolveFillMetric(cfg *config.Bubbletree) metric.Name {
	if fill := cfg.Fill.MetricName(); fill != "" {
		return fill
	}

	return metric.Name(ptrString(cfg.Size))
}

// resolveLabels converts the string labels flag to a bubbletree.LabelMode.
func (*BubbletreeCmd) resolveLabels(cfg *config.Bubbletree) bubbletree.LabelMode {
	if lbl := ptrString(cfg.Labels); lbl != "" {
		return bubbletree.LabelMode(lbl)
	}

	return bubbletree.LabelFoldersOnly
}
