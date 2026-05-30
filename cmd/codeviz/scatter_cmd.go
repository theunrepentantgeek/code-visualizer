package main

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	scatterviz "github.com/theunrepentantgeek/code-visualizer/internal/scatter"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

type ScatterCmd struct {
	TargetPath string `arg:"" help:"Path to directory to scan."`
	Output     string `help:"Output image file path (png, jpg, jpeg, svg)." required:"true" short:"o"`

	XAxis metric.Name `default:"" help:"Metric for X-axis position; run 'codeviz help-metrics' for available metrics." name:"x-axis" short:"x"` //nolint:revive,nolintlint // kong struct tags require long lines
	YAxis metric.Name `default:"" help:"Metric for Y-axis position; run 'codeviz help-metrics' for available metrics." name:"y-axis" short:"y"` //nolint:revive,nolintlint // kong struct tags require long lines
	Size  metric.Name `default:"" help:"Metric for disc size; run 'codeviz help-metrics' for available metrics." short:"s"`                     //nolint:revive,nolintlint // kong struct tags require long lines

	Fill   config.MetricSpec `help:"Fill colour: metric[,palette] (e.g. file-type,categorization)." optional:"" short:"f"` //nolint:revive,nolintlint // kong struct tags require long lines
	Border config.MetricSpec `help:"Border colour: metric[,palette] (e.g. file-lines,foliage)." optional:"" short:"b"`     //nolint:revive,nolintlint // kong struct tags require long lines

	Legend            string `default:"" enum:",top-left,top-center,top-right,center-right,bottom-right,bottom-center,bottom-left,center-left,none" help:"Legend position (default: bottom-right)." optional:""` //nolint:revive,nolintlint // kong struct tags require long lines
	LegendOrientation string `default:"" enum:",vertical,horizontal" help:"Legend orientation (auto-detected from position if omitted)." name:"legend-orientation" optional:""`                                  //nolint:revive,nolintlint // kong struct tags require long lines

	Width  int `default:"1920" help:"Image width in pixels."`
	Height int `default:"1080" help:"Image height in pixels."`

	Include            []filter.Rule `type:"filterrule" name:"include" help:"Include matching files (repeatable)." placeholder:"glob"`                 //nolint:revive,nolintlint // kong struct tags require long lines
	Exclude            []filter.Rule `type:"filterrule" name:"exclude" help:"Exclude matching files (repeatable)." placeholder:"glob"`                 //nolint:revive,nolintlint // kong struct tags require long lines
	IncludeBinaryFiles bool          `help:"Include binary files in the visualization (excluded by default)." name:"include-binary-files" optional:""` //nolint:revive,nolintlint // kong struct tags require long lines
}

func (c *ScatterCmd) Filters() []filter.Rule {
	return filter.Merge(c.Include, c.Exclude)
}

func (*ScatterCmd) Validate() error {
	return nil
}

func (*ScatterCmd) validateConfig(cfg *config.Scatter) error {
	if err := validateScatterAxisMetric("x-axis", cfg.XAxis); err != nil {
		return err
	}

	if err := validateScatterAxisMetric("y-axis", cfg.YAxis); err != nil {
		return err
	}

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

func validateScatterAxisMetric(label string, name *string) error {
	axis := ptrString(name)
	if _, ok := provider.GetDescriptor(metric.Name(axis)); !ok {
		return eris.Errorf("unknown %s metric %q; available metrics: %s", label, axis, formatMetricNames())
	}

	return nil
}

func (c *ScatterCmd) mergeConfigAndValidate(flags *Flags) error {
	if err := flags.Config.TryAutoLoad(c.Output); err != nil {
		return eris.Wrap(err, "auto-config load failed")
	}

	c.applyOverrides(flags.Config)

	return c.validateConfig(flags.Config.Scatter)
}

func (c *ScatterCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	state := &scatterviz.State{
		CommonState: stages.CommonState{
			TargetPath: c.TargetPath,
			Output:     c.Output,
			Flags:      toStagesFlags(flags),
			RootConfig: flags.Config,
			CLIFilters: c.Filters(),
		},
		Config:             flags.Config.Scatter,
		IncludeBinaryFiles: c.IncludeBinaryFiles,
	}

	_, err := pipeline.Run[*scatterviz.State](
		state,
		stages.ValidatePaths,
		stages.ExportConfig,
		stages.BuildFilterRules,
		scatterviz.ResolveMetrics,
		stages.ScanFilesystem,
		stages.CheckGitRequirement,
		stages.RunProviders,
		stages.FilterBinaryFiles,
		stages.ExportData,
		stages.ResolveDimensions,
		scatterviz.BuildInksStage,
		scatterviz.BuildLegendStage,
		scatterviz.LayoutStage,
		scatterviz.RenderStage,
		stages.WriteCanvas,
		scatterviz.LogResult,
	)

	return eris.Wrap(err, "scatter pipeline failed")
}

func (c *ScatterCmd) applyOverrides(cfg *config.Config) {
	cfg.OverrideWidth(c.Width)
	cfg.OverrideHeight(c.Height)

	if cfg.Scatter == nil {
		cfg.Scatter = &config.Scatter{}
	}

	cfg.Scatter.OverrideXAxis(string(c.XAxis))
	cfg.Scatter.OverrideYAxis(string(c.YAxis))
	cfg.Scatter.OverrideSize(string(c.Size))
	cfg.Scatter.OverrideFill(c.Fill)
	cfg.Scatter.OverrideBorder(c.Border)
	cfg.Scatter.OverrideLegend(c.Legend)
	cfg.Scatter.OverrideLegendOrientation(c.LegendOrientation)
}
