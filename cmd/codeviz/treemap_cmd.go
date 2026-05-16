package main

import (
	"log/slog"
	"strings"

	"github.com/rotisserie/eris"

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
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
)

type TreemapCmd struct {
	TargetPath string `arg:"" help:"Path to directory to scan."`
	Output     string `help:"Output image file path (png, jpg, jpeg, svg)." required:"true" short:"o"`

	Size metric.Name `default:"" help:"Metric for rectangle area; run 'codeviz help-metrics' for available metrics." short:"s"` //nolint:revive,nolintlint // kong struct tags require long lines

	Fill   config.MetricSpec `help:"Fill colour: metric[,palette] (e.g. file-type,categorization)." optional:"" short:"f"` //nolint:revive,nolintlint // kong struct tags require long lines
	Border config.MetricSpec `help:"Border colour: metric[,palette] (e.g. file-lines,foliage)." optional:"" short:"b"`     //nolint:revive,nolintlint // kong struct tags require long lines

	Legend            string `default:"" enum:",top-left,top-center,top-right,center-right,bottom-right,bottom-center,bottom-left,center-left,none" help:"Legend position (default: bottom-right)." optional:""` //nolint:revive // kong struct tags require long lines
	LegendOrientation string `default:"" enum:",vertical,horizontal" help:"Legend orientation (auto-detected from position if omitted)." name:"legend-orientation" optional:""`                                  //nolint:revive // kong struct tags require long lines

	Width  int `default:"1920" help:"Image width in pixels."`
	Height int `default:"1080" help:"Image height in pixels."`

	Filter             []string `help:"Filter rule: glob to include, !glob to exclude (repeatable, order-preserved)."`
	IncludeBinaryFiles bool     `help:"Include binary files in the visualization (excluded by default)." name:"include-binary-files" optional:""` //nolint:revive,nolintlint // kong struct tags require long lines
}

func (c *TreemapCmd) Validate() error {
	for _, f := range c.Filter {
		if _, err := filter.ParseFilterFlag(f); err != nil {
			return eris.Wrapf(err, "invalid filter %q", f)
		}
	}

	return nil
}

// validateConfig checks the effective configuration after all sources have been
// merged. Called from mergeConfigAndValidate() after TryAutoLoad + applyOverrides.
func (*TreemapCmd) validateConfig(cfg *config.Treemap) error {
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

func formatMetricNames() string {
	names := provider.Names()
	strs := make([]string, len(names))

	for i, n := range names {
		strs[i] = string(n)
	}

	return strings.Join(strs, ", ")
}

func collectRequestedMetrics(size metric.Name, fill, border *config.MetricSpec) []metric.Name {
	seen := map[metric.Name]bool{size: true}
	names := []metric.Name{size}

	for _, spec := range []*config.MetricSpec{fill, border} {
		if spec != nil && spec.Metric != "" {
			if !seen[spec.Metric] {
				seen[spec.Metric] = true
				names = append(names, spec.Metric)
			}
		}
	}

	return names
}

// specMetric returns the metric name from a *MetricSpec, or "" if nil.
func specMetric(s *config.MetricSpec) metric.Name {
	if s == nil {
		return ""
	}

	return s.Metric
}

// specPalette returns the palette name from a *MetricSpec, or "" if nil.
func specPalette(s *config.MetricSpec) palette.PaletteName {
	if s == nil {
		return ""
	}

	return s.Palette
}

// mergeConfigAndValidate loads the config file, merges CLI overrides on top,
// and validates the effective configuration. Called at the start of Run().
func (c *TreemapCmd) mergeConfigAndValidate(flags *Flags) error {
	if err := flags.Config.TryAutoLoad(c.Output); err != nil {
		return eris.Wrap(err, "auto-config load failed")
	}

	c.applyOverrides(flags.Config)

	return c.validateConfig(flags.Config.Treemap)
}

//nolint:dupl,revive,cyclop,funlen // Run methods share workflow structure across visualization commands
func (c *TreemapCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	cfg := flags.Config.Treemap
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
	fillPaletteName := resolveFillPalette(cfg.Fill, fillMetric)

	filterRules := stages.BuildFilterRulesHelper(flags.Config, c.Filter)

	slog.Info("Scanning filesystem", "path", c.TargetPath)

	scanProg, stopScanTicker := buildScanProgress(flags)

	root, err := scan.Scan(c.TargetPath, filterRules, scanProg)

	stopScanTicker()

	if err != nil {
		return eris.Wrap(err, "scan failed")
	}

	// Collect all requested metrics and run providers
	requested := collectRequestedMetrics(size, cfg.Fill, cfg.Border)

	// Check git requirement before running providers
	if err := stages.CheckGitRequirementHelper(c.TargetPath, requested); err != nil {
		return eris.Wrap(err, "git requirement check failed")
	}

	slog.Info("Calculating metrics")

	metricProg, stopMetricTicker := buildMetricProgress(flags, model.CountFiles(root))

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

func (c *TreemapCmd) renderAndLog(
	root *model.Directory,
	cfg *config.Treemap,
	width, height int,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
) error {
	size := metric.Name(ptrString(cfg.Size))
	files, dirs := stages.CountAll(root)

	slog.Info("Rendering image", "output", c.Output, "width", width, "height", height)

	// Build inks first — legend uses the same Ink objects
	borderName, borderPaletteName := resolveBorderMetricAndPalette(cfg.Border)
	inks := buildTreemapInks(root, fillMetric, fillPaletteName, borderName, borderPaletteName)

	// Build legend config from the Inks
	legendPos, legendOrient := legend.ResolveOptions(ptrString(cfg.Legend), ptrString(cfg.LegendOrientation))
	legendConfig := legend.Build(
		legendPos, legendOrient,
		inks.fill, fillMetric,
		inks.border, borderName,
		size,
	)

	// Reserve space and layout
	layoutW, layoutH := legend.ReserveAndLayout(legendConfig, width, height)

	rects := treemap.Layout(root, layoutW, layoutH, size)

	if layoutW < width || layoutH < height {
		if legendConfig != nil {
			wReduce, hReduce := legendConfig.ReserveSpace()
			dx, dy := legend.LayoutOffset(legendConfig, wReduce, hReduce)
			treemap.OffsetRects(&rects, dx, dy)
		}
	}

	cv := renderTreemapToCanvas(rects, root, width, height, inks)

	if legendConfig != nil {
		cv.SetLegend(*legendConfig)
	}

	slog.Debug("rendering", "width", width, "height", height, "output", c.Output)

	if err := cv.Render(c.Output); err != nil {
		return eris.Wrap(err, "render failed")
	}

	slog.Info(
		"Rendered treemap",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"width", width,
		"height", height,
		"size_metric", string(size),
		"fill_metric", string(fillMetric),
		"fill_palette", string(fillPaletteName),
		"border_metric", string(borderName),
		"border_palette", string(borderPaletteName),
	)

	return nil
}

// applyOverrides writes non-zero CLI flag values on top of the config layer.
// Zero-valued CLI fields are transparent — the config value passes through unchanged.
func (c *TreemapCmd) applyOverrides(cfg *config.Config) {
	cfg.OverrideWidth(c.Width)
	cfg.OverrideHeight(c.Height)
	cfg.Treemap.OverrideSize(string(c.Size))
	cfg.Treemap.OverrideFill(c.Fill)
	cfg.Treemap.OverrideBorder(c.Border)
	cfg.Treemap.OverrideLegend(c.Legend)
	cfg.Treemap.OverrideLegendOrientation(c.LegendOrientation)
}

// ptrString safely dereferences a *string, returning "" if nil.
func ptrString(p *string) string {
	if p == nil {
		return ""
	}

	return *p
}

// ptrInt safely dereferences a *int, returning fallback if nil.
func ptrInt(p *int, fallback int) int {
	if p == nil {
		return fallback
	}

	return *p
}

func (*TreemapCmd) resolveFillMetric(cfg *config.Treemap) metric.Name {
	if fill := specMetric(cfg.Fill); fill != "" {
		return fill
	}

	return metric.Name(ptrString(cfg.Size))
}
