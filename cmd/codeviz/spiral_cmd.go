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

	Size metric.Name `default:"" help:"Metric for disc size; run 'codeviz help metrics' for available metrics." short:"s"` //nolint:revive,nolintlint // kong struct tags require long lines

	Fill   config.MetricSpec `help:"Fill colour: metric[,palette] (e.g. file-type,categorization)." optional:"" short:"f"` //nolint:revive,nolintlint // kong struct tags require long lines
	Border config.MetricSpec `help:"Border colour: metric[,palette] (e.g. file-lines,foliage)." optional:"" short:"b"`     //nolint:revive,nolintlint // kong struct tags require long lines

	Labels string `help:"Label mode: all, laps, or none." enum:",all,laps,none" default:""`

	Legend            string `default:"" enum:",top-left,top-center,top-right,center-right,bottom-right,bottom-center,bottom-left,center-left,none" help:"Legend position (default: bottom-right)." optional:""` //nolint:revive,nolintlint // kong struct tags require long lines
	LegendOrientation string `default:"" enum:",vertical,horizontal" help:"Legend orientation (auto-detected from position if omitted)." name:"legend-orientation" optional:""`                                  //nolint:revive,nolintlint // kong struct tags require long lines

	Width  int `default:"1920" help:"Canvas width in pixels."`
	Height int `default:"1920" help:"Canvas height in pixels."`

	Title      string `default:"" help:"Override title text on the generated image." optional:""`
	Footer     string `default:"" help:"Override footer text on the generated image." optional:""`
	HideFooter bool   `default:"false" help:"Suppress the attribution footer." name:"hide-footer" optional:""`

	Include            []filter.Rule `type:"filterrule" name:"include" help:"Include matching files (repeatable)." placeholder:"glob"`                 //nolint:revive,nolintlint // kong struct tags require long lines
	Exclude            []filter.Rule `type:"filterrule" name:"exclude" help:"Exclude matching files (repeatable)." placeholder:"glob"`                 //nolint:revive,nolintlint // kong struct tags require long lines
	IncludeBinaryFiles bool          `help:"Include binary files in the visualization (excluded by default)." name:"include-binary-files" optional:""` //nolint:revive,nolintlint // kong struct tags require long lines
}

func (c *SpiralCmd) Filters() []filter.Rule {
	return filter.Merge(c.Include, c.Exclude)
}

func (*SpiralCmd) Validate() error {
	return nil
}

// validateConfig checks the effective configuration after all sources have been
// merged. Called from mergeConfigAndValidate() after TryAutoLoad + applyOverrides.
func (*SpiralCmd) validateConfig(cfg *config.Spiral) error {
	size := ptrString(cfg.Size)
	if size != "" {
		d, ok := provider.GetBase(metric.Name(size))
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

	common := &stages.CommonState{
		TargetPath:         c.TargetPath,
		Output:             c.Output,
		Flags:              toStagesFlags(flags),
		RootConfig:         flags.Config,
		VizName:            "spiral",
		CLIFilters:         c.Filters(),
		IncludeBinaryFiles: c.IncludeBinaryFiles,
	}
	cfg := flags.Config.Spiral
	viz := &spiral.State{}

	s := pipeline.NewState(common, cfg, viz)

	pipeline.ApplyFuncX(s, stages.ValidatePaths)
	pipeline.ApplyFuncX(s, stages.ExportConfig)
	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(s, spiral.ResolveMetrics)
	pipeline.ApplyFuncX(s, stages.ScanFilesystem)
	pipeline.ApplyFuncX(s, stages.CheckGitRequirement)
	pipeline.ApplyFuncX(s, stages.RunProviders)
	pipeline.ApplyFuncX(s, stages.PopulateDeclarations)
	pipeline.ApplyFuncX(s, stages.RunAggregations)
	pipeline.ApplyFuncX(s, stages.FilterBinaryFiles)
	pipeline.ApplyFuncX(s, stages.ExportData)
	pipeline.ApplyFuncX(s, stages.LoadGitHistory)
	pipeline.ApplyFuncX(s, stages.GroupGitHistoryByFile)
	pipeline.ApplyFuncX(s, stages.ExtractFileHistory)
	pipeline.ApplyFuncX(s, stages.ResolveDimensions)
	pipeline.ApplyFuncX(s, stages.InitDrawingBounds)
	pipeline.ApplyFuncX(s, stages.ReserveTitleBounds)
	pipeline.ApplyFuncX(s, stages.ReserveFooterBounds)
	pipeline.ApplyFuncXY(s, spiral.BuildTimeBucketsStage)
	pipeline.ApplyFuncXY(s, spiral.AggregateBucketMetricsStage)
	pipeline.ApplyFuncXY(s, spiral.BuildInksStage)
	pipeline.ApplyFuncXY(s, spiral.BuildLegendStage)
	pipeline.ApplyFuncXY(s, spiral.LayoutStage)
	pipeline.ApplyFuncXY(s, spiral.RenderStage)
	pipeline.ApplyFuncX(s, stages.ApplyTitle)
	pipeline.ApplyFuncX(s, stages.ApplyFooter)
	pipeline.ApplyFuncX(s, stages.WriteCanvas)
	pipeline.ApplyFuncXY(s, spiral.LogResult)

	return eris.Wrap(s.Err(), "spiral pipeline failed")
}

// applyOverrides writes non-zero CLI flag values on top of the config layer.
// Zero-valued CLI fields are transparent — the config value passes through unchanged.
func (c *SpiralCmd) applyOverrides(cfg *config.Config) {
	cfg.OverrideWidth(c.Width)
	cfg.OverrideHeight(c.Height)
	cfg.OverrideTitleText(c.Title)
	cfg.OverrideFooterText(c.Footer)
	cfg.OverrideHideFooter(c.HideFooter)

	if cfg.Spiral == nil {
		cfg.Spiral = &config.Spiral{}
	}

	cfg.Spiral.OverrideResolution(c.Resolution)
	cfg.Spiral.OverrideSize(string(c.Size))
	cfg.Spiral.OverrideFill(c.Fill)
	cfg.Spiral.OverrideBorder(c.Border)
	cfg.Spiral.OverrideLabels(c.Labels)
	cfg.OverrideLegendPosition(c.Legend)
	cfg.OverrideLegendOrientation(c.LegendOrientation)
}
