package main

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	scatterviz "github.com/theunrepentantgeek/code-visualizer/internal/scatter"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

type ScatterCmd struct {
	TargetPath string `arg:"" help:"Path to directory to scan."`
	Output     string `help:"Output image file path (png, jpg, jpeg, svg)." required:"true" short:"o"`

	XAxis metric.Name `default:"" help:"Metric for X-axis position; run 'codeviz help metrics' for available metrics." name:"x-axis" short:"x"` //nolint:revive,nolintlint // kong struct tags require long lines
	YAxis metric.Name `default:"" help:"Metric for Y-axis position; run 'codeviz help metrics' for available metrics." name:"y-axis" short:"y"` //nolint:revive,nolintlint // kong struct tags require long lines
	Size  metric.Name `default:"" help:"Metric for disc size; run 'codeviz help metrics' for available metrics." short:"s"`                     //nolint:revive,nolintlint // kong struct tags require long lines

	XScale string `default:"" enum:",linear,log" help:"X-axis scale (linear or log)." name:"x-scale"` //nolint:revive,nolintlint // kong struct tags require long lines
	YScale string `default:"" enum:",linear,log" help:"Y-axis scale (linear or log)." name:"y-scale"` //nolint:revive,nolintlint // kong struct tags require long lines

	Fill   config.MetricSpec `help:"Fill colour: metric[,palette] (e.g. file-type,categorization)." optional:"" short:"f"` //nolint:revive,nolintlint // kong struct tags require long lines
	Border config.MetricSpec `help:"Border colour: metric[,palette] (e.g. file-lines,foliage)." optional:"" short:"b"`     //nolint:revive,nolintlint // kong struct tags require long lines

	Legend            string `default:"" enum:",top-left,top-center,top-right,center-right,bottom-right,bottom-center,bottom-left,center-left,none" help:"Legend position (default: bottom-right)." optional:""` //nolint:revive,nolintlint // kong struct tags require long lines
	LegendOrientation string `default:"" enum:",vertical,horizontal" help:"Legend orientation (auto-detected from position if omitted)." name:"legend-orientation" optional:""`                                  //nolint:revive,nolintlint // kong struct tags require long lines

	Width  int `default:"1920" help:"Image width in pixels."`
	Height int `default:"1080" help:"Image height in pixels."`

	Title      string `default:"" help:"Override title text on the generated image." optional:""`
	Footer     string `default:"" help:"Override footer text on the generated image." optional:""`
	HideFooter bool   `default:"false" help:"Suppress the attribution footer." name:"hide-footer" optional:""`

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

	if err := validateNumericMetric("size", metric.Name(ptrString(cfg.Size))); err != nil {
		return err
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
	return validateMetricExists(label, metric.Name(ptrString(name)))
}

func (c *ScatterCmd) mergeConfigAndValidate(flags *Flags) error {
	if err := flags.Config.TryAutoLoad(c.Output); err != nil {
		return eris.Wrap(err, "auto-config load failed")
	}

	c.applyOverrides(flags.Config)

	return c.validateConfig(flags.Config.Scatter)
}

//nolint:dupl // pipeline wiring is structurally similar across commands but not refactorable
func (c *ScatterCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	common := &stages.CommonState{
		TargetPath:         c.TargetPath,
		Output:             c.Output,
		Flags:              toStagesFlags(flags),
		RootConfig:         flags.Config,
		VizName:            "scatter",
		CLIFilters:         c.Filters(),
		IncludeBinaryFiles: c.IncludeBinaryFiles,
	}
	cfg := flags.Config.Scatter
	viz := &scatterviz.State{}

	s := pipeline.NewState(common, cfg, viz)

	pipeline.ApplyFuncX(s, stages.ValidatePaths)
	pipeline.ApplyFuncX(s, stages.ExportConfig)
	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(s, scatterviz.ResolveMetrics)

	scatterviz.AcquireData(s)
	scatterviz.RenderPipeline(s)

	return eris.Wrap(s.Err(), "scatter pipeline failed")
}

func (c *ScatterCmd) applyOverrides(cfg *config.Config) {
	cfg.OverrideWidth(c.Width)
	cfg.OverrideHeight(c.Height)
	cfg.OverrideTitleText(c.Title)
	cfg.OverrideFooterText(c.Footer)
	cfg.OverrideHideFooter(c.HideFooter)

	if cfg.Scatter == nil {
		cfg.Scatter = &config.Scatter{}
	}

	cfg.Scatter.OverrideXAxis(string(c.XAxis))
	cfg.Scatter.OverrideYAxis(string(c.YAxis))
	cfg.Scatter.OverrideSize(string(c.Size))
	cfg.Scatter.OverrideXScale(c.XScale)
	cfg.Scatter.OverrideYScale(c.YScale)
	cfg.Scatter.OverrideFill(c.Fill)
	cfg.Scatter.OverrideBorder(c.Border)
	cfg.OverrideLegendPosition(c.Legend)
	cfg.OverrideLegendOrientation(c.LegendOrientation)
}
