package main

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
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

	Legend            string `default:"" enum:",top-left,top-center,top-right,center-right,bottom-right,bottom-center,bottom-left,center-left,none" help:"Legend position (default: bottom-right)." optional:""` //nolint:revive,nolintlint // kong struct tags require long lines
	LegendOrientation string `default:"" enum:",vertical,horizontal" help:"Legend orientation (auto-detected from position if omitted)." name:"legend-orientation" optional:""`                                  //nolint:revive,nolintlint // kong struct tags require long lines

	Width  int `default:"1920" help:"Canvas width in pixels."`
	Height int `default:"1920" help:"Canvas height in pixels."`

	Filters            []filter.Rule `kong:"-"`
	Include            []filter.Rule `type:"filterrule" name:"include" help:"Include matching files (repeatable)." placeholder:"glob"`
	Exclude            []filter.Rule `type:"filterrule" name:"exclude" help:"Exclude matching files (repeatable)." placeholder:"glob"`
	IncludeBinaryFiles bool          `help:"Include binary files in the visualization (excluded by default)." name:"include-binary-files" optional:""` //nolint:revive,nolintlint // kong struct tags require long lines
}

func (c *SpiralCmd) Validate() error {
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

	state := &spiral.State{
		CommonState: stages.CommonState{
			TargetPath: c.TargetPath,
			Output:     c.Output,
			Flags:      toStagesFlags(flags),
			RootConfig: flags.Config,
			CLIFilters: c.Filters,
		},
		Config:             flags.Config.Spiral,
		IncludeBinaryFiles: c.IncludeBinaryFiles,
	}

	_, err := pipeline.Run[*spiral.State](
		state,
		stages.ValidatePaths,
		stages.ExportConfig,
		stages.BuildFilterRules,
		spiral.ResolveMetrics,
		stages.ScanFilesystem,
		stages.CheckGitRequirement,
		stages.RunProviders,
		stages.FilterBinaryFiles,
		stages.ExportData,
		stages.LoadGitHistory,
		stages.GroupGitHistoryByFile,
		stages.ExtractFileHistory,
		stages.ResolveDimensions,
		spiral.BuildTimeBucketsStage,
		spiral.AggregateBucketMetricsStage,
		spiral.BuildInksStage,
		spiral.BuildLegendStage,
		spiral.LayoutStage,
		spiral.RenderStage,
		stages.WriteCanvas,
		spiral.LogResult,
	)

	return eris.Wrap(err, "spiral pipeline failed")
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
