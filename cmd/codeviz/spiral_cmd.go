package main

import (
	"log/slog"

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
	"github.com/theunrepentantgeek/code-visualizer/internal/spiral"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

type SpiralCmd struct {
	TargetPath string `arg:"" help:"Path to directory to scan."`
	Output     string `help:"Output image file path (png, jpg, jpeg, svg)." required:"true" short:"o"`

	Resolution string `short:"r" help:"Time resolution (hourly or daily)." enum:",hourly,daily" default:""`

	Size metric.Name `default:"" help:"Metric for disc size; run 'codeviz help-metrics' for available metrics." short:"s"` //nolint:revive,nolintlint // kong struct tags require long lines

	Fill   config.MetricSpec `help:"Fill colour: metric[,palette] (e.g. file-type,categorization)." optional:"" short:"f"` //nolint:revive,nolintlint // kong struct tags require long lines
	Border config.MetricSpec `help:"Border colour: metric[,palette] (e.g. file-lines,foliage)." optional:"" short:"b"`     //nolint:revive,nolintlint // kong struct tags require long lines

	Labels string `help:"Label mode: all, laps, or none." enum:",all,laps,none" default:""`

	Legend            string `default:"" enum:",top-left,top-center,top-right,center-right,bottom-right,bottom-center,bottom-left,center-left,none" help:"Legend position (default: bottom-right)." optional:""` //nolint:revive // kong struct tags require long lines
	LegendOrientation string `default:"" enum:",vertical,horizontal" help:"Legend orientation (auto-detected from position if omitted)." name:"legend-orientation" optional:""`                                  //nolint:revive // kong struct tags require long lines

	Width  int `default:"1920" help:"Canvas width in pixels."`
	Height int `default:"1920" help:"Canvas height in pixels."`

	Filter             []string `help:"Filter rule: glob to include, !glob to exclude (repeatable, order-preserved)."`                            //nolint:revive // kong struct tags require long lines
	IncludeBinaryFiles bool     `help:"Include binary files in the visualization (excluded by default)." name:"include-binary-files" optional:""` //nolint:revive // kong struct tags require long lines
}

func (c *SpiralCmd) Validate() error {
	for _, f := range c.Filter {
		if _, err := filter.ParseFilterFlag(f); err != nil {
			return eris.Wrapf(err, "invalid filter %q", f)
		}
	}

	return nil
}

// validateConfig checks the effective configuration after all sources have been
// merged. Called from mergeConfigAndValidate() after TryAutoLoad + applyOverrides.
func (*SpiralCmd) validateConfig(cfg *config.Spiral) error {
	size := ptrString(cfg.Size)
	if size != "" {
		d, ok := provider.GetDescriptor(metric.Name(size))
		if !ok {
			return eris.Errorf("unknown size metric %q; available metrics: %s", size, formatMetricNames())
		}

		if d.Kind != metric.Quantity && d.Kind != metric.Measure {
			return eris.Errorf("size metric must be numeric, got %q (kind: %d)", size, d.Kind)
		}
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
func (c *SpiralCmd) mergeConfigAndValidate(flags *Flags) error {
	if err := flags.Config.TryAutoLoad(c.Output); err != nil {
		return eris.Wrap(err, "auto-config load failed")
	}

	c.applyOverrides(flags.Config)

	return c.validateConfig(flags.Config.Spiral)
}

func (c *SpiralCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	cfg := flags.Config.Spiral

	if err := stages.ValidatePathsHelper(c.TargetPath, c.Output); err != nil {
		return eris.Wrap(err, "path validation failed")
	}

	if flags.ExportConfig != "" {
		if err := flags.Config.Save(flags.ExportConfig); err != nil {
			return eris.Wrap(err, "failed to save config")
		}
	}

	root, err := c.scanAndRunProviders(flags, cfg)
	if err != nil {
		return err
	}

	buckets, err := c.buildTimeBuckets(flags, root, cfg)
	if err != nil {
		return err
	}

	sizeMetric := metric.Name(ptrString(cfg.Size))
	fillMetric := cfg.Fill.MetricName()
	borderMetric := cfg.Border.MetricName()
	spiral.AggregateBucketMetrics(buckets, sizeMetric, fillMetric, borderMetric)

	return c.layoutAndRender(flags, cfg, root, buckets)
}

func (c *SpiralCmd) scanAndRunProviders(flags *Flags, cfg *config.Spiral) (*model.Directory, error) {
	filterRules := stages.BuildFilterRulesHelper(flags.Config, c.Filter)

	slog.Info("Scanning filesystem", "path", c.TargetPath)

	scanProg, stopScanTicker := stages.BuildScanProgress(toStagesFlags(flags))

	root, err := scan.Scan(c.TargetPath, filterRules, scanProg)

	stopScanTicker()

	if err != nil {
		return nil, eris.Wrap(err, "scan failed")
	}

	requested := c.collectSpiralMetrics(cfg)

	slog.Info("Calculating metrics")

	metricProg, stopMetricTicker := stages.BuildMetricProgress(toStagesFlags(flags), model.CountFiles(root))

	if err := provider.Run(root, requested, metricProg); err != nil {
		stopMetricTicker()

		return nil, eris.Wrap(err, "failed to load metrics")
	}

	stopMetricTicker()

	if !c.IncludeBinaryFiles {
		if err := stages.FilterBinaryFilesHelper(root); err != nil {
			return nil, eris.Wrap(err, "binary file filter failed")
		}
	}

	if err := export.Export(root, requested, flags.ExportData); err != nil {
		return nil, eris.Wrap(err, "failed to export data")
	}

	return root, nil
}

func (c *SpiralCmd) buildTimeBuckets(
	flags *Flags,
	root *model.Directory,
	cfg *config.Spiral,
) ([]spiral.TimeBucket, error) {
	if err := stages.CheckGitRepoHelper(c.TargetPath); err != nil {
		return nil, eris.Wrap(err, "git requirement check failed")
	}

	slog.Info("Loading commit history")

	histProg, stopHistTicker := stages.BuildHistoryProgress(toStagesFlags(flags))

	records, err := loadCommitHistory(root, histProg)

	stopHistTicker()

	if err != nil {
		return nil, eris.Wrap(err, "failed to load commit history")
	}

	if len(records) == 0 {
		return nil, eris.New("no commit history found; spiral requires git commits")
	}

	startTime, endTime := records[0].Timestamp, records[0].Timestamp
	for _, r := range records[1:] {
		if r.Timestamp.Before(startTime) {
			startTime = r.Timestamp
		}

		if r.Timestamp.After(endTime) {
			endTime = r.Timestamp
		}
	}

	resolution := c.resolveResolution(cfg)

	buckets := spiral.BuildTimeBuckets(resolution, startTime, endTime)
	if len(buckets) == 0 {
		return nil, eris.New("no time buckets created from commit time range")
	}

	fileHistory := make(map[*model.File][]stages.CommitRef, len(records))
	for _, rec := range records {
		fileHistory[rec.File] = append(fileHistory[rec.File], stages.CommitRef{When: rec.Timestamp})
	}

	spiral.AssignFilesToBuckets(buckets, fileHistory)

	return buckets, nil
}

func (c *SpiralCmd) layoutAndRender(
	flags *Flags,
	cfg *config.Spiral,
	root *model.Directory,
	buckets []spiral.TimeBucket,
) error {
	width := ptrInt(flags.Config.Width)
	height := ptrInt(flags.Config.Height)
	resolution := c.resolveResolution(cfg)
	labels := c.resolveLabels(cfg)

	layout := spiral.Layout(buckets, width, height, resolution, labels)
	maxDisc := spiral.MaxDiscRadius(len(buckets), width, height, resolution)
	spiral.ApplyDiscSizes(layout.Nodes, buckets, maxDisc)

	fillMetric := c.resolveFillMetric(cfg)
	fillPaletteName := stages.ResolveFillPalette(cfg.Fill, fillMetric)
	borderMetric, borderPaletteName := stages.ResolveBorderMetricAndPalette(cfg.Border)

	inks := spiral.BuildInks(buckets, fillMetric, fillPaletteName, borderMetric, borderPaletteName)

	slog.Info("Rendering image", "output", c.Output, "width", width, "height", height)

	cv := spiral.RenderToCanvas(layout, buckets, width, height, inks)

	legendPos, legendOrient := legend.ResolveOptions(ptrString(cfg.Legend), ptrString(cfg.LegendOrientation))
	legendConfig := legend.Build(
		legendPos, legendOrient,
		inks.Fill, fillMetric,
		inks.Border, borderMetric,
		metric.Name(ptrString(cfg.Size)),
	)

	if legendConfig != nil {
		cv.SetLegend(*legendConfig)
	}

	if err := cv.Render(c.Output); err != nil {
		return eris.Wrap(err, "render failed")
	}

	sizeMetric := metric.Name(ptrString(cfg.Size))
	c.logRendered(root, width, height, sizeMetric, fillMetric, fillPaletteName, borderMetric, borderPaletteName)

	return nil
}

func (*SpiralCmd) logRendered(
	root *model.Directory,
	width, height int,
	sizeMetric, fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) {
	files, dirs := stages.CountAll(root)

	slog.Info(
		"Rendered spiral",
		"files", files,
		"directories", dirs,
		"width", width,
		"height", height,
		"size_metric", string(sizeMetric),
		"fill_metric", string(fillMetric),
		"fill_palette", string(fillPaletteName),
		"border_metric", string(borderMetric),
		"border_palette", string(borderPaletteName),
	)
}

// applyOverrides writes non-zero CLI flag values on top of the config layer.
// Zero-valued CLI fields are transparent — the config value passes through unchanged.
func (c *SpiralCmd) applyOverrides(cfg *config.Config) {
	cfg.OverrideWidth(c.Width)
	cfg.OverrideHeight(c.Height)

	if cfg.Spiral == nil {
		cfg.Spiral = &config.Spiral{}
	}

	cfg.Spiral.OverrideResolution(c.Resolution)
	cfg.Spiral.OverrideSize(string(c.Size))
	cfg.Spiral.OverrideFill(c.Fill)
	cfg.Spiral.OverrideBorder(c.Border)
	cfg.Spiral.OverrideLabels(c.Labels)
	cfg.Spiral.OverrideLegend(c.Legend)
	cfg.Spiral.OverrideLegendOrientation(c.LegendOrientation)
}

// collectSpiralMetrics gathers all metrics requested by the spiral configuration.
// Unlike other commands, size is optional — when omitted, disc size defaults to commit count.
func (*SpiralCmd) collectSpiralMetrics(cfg *config.Spiral) []metric.Name {
	size := metric.Name(ptrString(cfg.Size))
	if size != "" {
		return stages.CollectRequestedMetrics(size, cfg.Fill, cfg.Border)
	}

	seen := map[metric.Name]bool{}

	var names []metric.Name

	for _, spec := range []*config.MetricSpec{cfg.Fill, cfg.Border} {
		if spec != nil && spec.Metric != "" && !seen[spec.Metric] {
			seen[spec.Metric] = true
			names = append(names, spec.Metric)
		}
	}

	return names
}

func (*SpiralCmd) resolveResolution(cfg *config.Spiral) spiral.Resolution {
	if r := ptrString(cfg.Resolution); r == "hourly" {
		return spiral.Hourly
	}

	return spiral.Daily
}

func (*SpiralCmd) resolveLabels(cfg *config.Spiral) spiral.LabelMode {
	if lbl := ptrString(cfg.Labels); lbl != "" {
		return spiral.LabelMode(lbl)
	}

	return spiral.LabelLaps
}

func (*SpiralCmd) resolveFillMetric(cfg *config.Spiral) metric.Name {
	return cfg.Fill.MetricName()
}
