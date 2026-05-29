package main

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/radialtree"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

type RadialCmd struct {
	TargetPath string `arg:"" help:"Path to directory to scan."`
	Output     string `help:"Output image file path (png, jpg, jpeg, svg)." required:"true" short:"o"`

	DiscSize metric.Name `default:"" help:"Metric for disc size; run 'codeviz help-metrics' for available metrics." short:"d"` //nolint:revive,nolintlint // kong struct tags require long lines

	Fill   config.MetricSpec `help:"Fill colour: metric[,palette] (e.g. file-type,categorization)." optional:"" short:"f"` //nolint:revive,nolintlint // kong struct tags require long lines
	Border config.MetricSpec `help:"Border colour: metric[,palette] (e.g. file-lines,foliage)." optional:"" short:"b"`     //nolint:revive,nolintlint // kong struct tags require long lines

	Labels string `enum:",all,folders,none" default:"" help:"Labels to display: all, folders, or none."`

	Legend            string `default:"" enum:",top-left,top-center,top-right,center-right,bottom-right,bottom-center,bottom-left,center-left,none" help:"Legend position (default: bottom-right)." optional:""` //nolint:revive,nolintlint // kong struct tags require long lines
	LegendOrientation string `default:"" enum:",vertical,horizontal" help:"Legend orientation (auto-detected from position if omitted)." name:"legend-orientation" optional:""`                                  //nolint:revive,nolintlint // kong struct tags require long lines

	Width  int `default:"1920" help:"Image width in pixels."`
	Height int `default:"1920" help:"Image height in pixels."`

	Filters            []filter.Rule `kong:"-"`
	Include            []filter.Rule `type:"filterrule" name:"include" help:"Include matching files (repeatable)." placeholder:"glob"`                 //nolint:revive,nolintlint // kong struct tags require long lines
	Exclude            []filter.Rule `type:"filterrule" name:"exclude" help:"Exclude matching files (repeatable)." placeholder:"glob"`                 //nolint:revive,nolintlint // kong struct tags require long lines
	IncludeBinaryFiles bool          `help:"Include binary files in the visualization (excluded by default)." name:"include-binary-files" optional:""` //nolint:revive // kong struct tags require long lines
}

func (*RadialCmd) Validate() error {
	return nil
}

// validateConfig checks the effective configuration after all sources have been
// merged. Called from mergeConfigAndValidate() after TryAutoLoad + applyOverrides.
func (*RadialCmd) validateConfig(cfg *config.Radial) error {
	discSize := ptrString(cfg.DiscSize)

	d, ok := provider.GetDescriptor(metric.Name(discSize))
	if !ok {
		return eris.Errorf("unknown disc-size metric %q; available metrics: %s", discSize, formatMetricNames())
	}

	if d.Kind != metric.Quantity && d.Kind != metric.Measure {
		return eris.Errorf("disc-size metric must be numeric, got %q (kind: %d)", discSize, d.Kind)
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

	state := &radialtree.State{
		CommonState: stages.CommonState{
			TargetPath: c.TargetPath,
			Output:     c.Output,
			Flags:      toStagesFlags(flags),
			RootConfig: flags.Config,
			CLIFilters: c.Filters,
		},
		Config:             flags.Config.Radial,
		IncludeBinaryFiles: c.IncludeBinaryFiles,
	}

	_, err := pipeline.Run[*radialtree.State](
		state,
		stages.ValidatePaths,
		stages.ExportConfig,
		stages.BuildFilterRules,
		radialtree.ResolveMetrics,
		stages.ScanFilesystem,
		stages.CheckGitRequirement,
		stages.RunProviders,
		stages.FilterBinaryFiles,
		stages.ExportData,
		stages.ResolveDimensions,
		radialtree.BuildInksStage,
		radialtree.BuildLegendStage,
		radialtree.LayoutStage,
		radialtree.RenderStage,
		stages.WriteCanvas,
		radialtree.LogResult,
	)

	return eris.Wrap(err, "radialtree pipeline failed")
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
