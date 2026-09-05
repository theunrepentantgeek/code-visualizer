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
	vizpkg "github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

type ScatterCmd struct {
	TargetPath string `arg:"" help:"Path to directory to scan."`
	Output     string `help:"Output image file path (png, jpg, jpeg, svg)." required:"true" short:"o"`
	From       string `help:"Git history lower bound (tag, commit ID, or date; prefixes: tag:, sha:, date:)." name:"from" optional:""`  //nolint:revive,nolintlint // kong struct tags require long lines
	Until      string `help:"Git history upper bound (tag, commit ID, or date; prefixes: tag:, sha:, date:)." name:"until" optional:""` //nolint:revive,nolintlint // kong struct tags require long lines

	XAxis metric.Name `default:"" help:"Metric for X-axis position; run 'codeviz help metrics' for available metrics." name:"x-axis" short:"x"` //nolint:revive,nolintlint // kong struct tags require long lines
	YAxis metric.Name `default:"" help:"Metric for Y-axis position; run 'codeviz help metrics' for available metrics." name:"y-axis" short:"y"` //nolint:revive,nolintlint // kong struct tags require long lines
	Size  metric.Name `default:"" help:"Metric for disc size; run 'codeviz help metrics' for available metrics." short:"s"`                     //nolint:revive,nolintlint // kong struct tags require long lines

	XScale string `default:"" enum:",linear,log" help:"X-axis scale (linear or log)." name:"x-scale"` //nolint:revive,nolintlint // kong struct tags require long lines
	YScale string `default:"" enum:",linear,log" help:"Y-axis scale (linear or log)." name:"y-scale"` //nolint:revive,nolintlint // kong struct tags require long lines

	Fill   config.MetricSpec `help:"Fill colour: metric[,palette] (e.g. file-type,categorization)." optional:"" short:"f"` //nolint:revive,nolintlint // kong struct tags require long lines
	Border config.MetricSpec `help:"Border colour: metric[,palette] (e.g. file-lines,foliage)." optional:"" short:"b"`     //nolint:revive,nolintlint // kong struct tags require long lines

	Grain string `enum:",file,directory" default:"" help:"Granularity of nodes shown: file (default) or directory."`

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
	level, err := scatterMetricLevel(cfg.Grain)
	if err != nil {
		return err
	}

	if err := validateScatterAxisMetric("x-axis", cfg.XAxis, level); err != nil {
		return err
	}

	if err := validateScatterAxisMetric("y-axis", cfg.YAxis, level); err != nil {
		return err
	}

	if err := validateScatterNumericMetric("size", metric.Name(ptrString(cfg.Size)), level); err != nil {
		return err
	}

	return validateScatterColours(cfg, level)
}

func validateScatterColours(cfg *config.Scatter, level metric.MetricLevel) error {
	if err := cfg.Fill.Validate("fill"); err != nil {
		return eris.Wrap(err, "invalid fill spec")
	}

	if err := cfg.Border.Validate("border"); err != nil {
		return eris.Wrap(err, "invalid border spec")
	}

	specs := []struct {
		label string
		name  metric.Name
	}{
		{label: "fill", name: cfg.Fill.MetricName()},
		{label: "border", name: cfg.Border.MetricName()},
	}

	for _, spec := range specs {
		if spec.name == "" {
			continue
		}

		if _, err := resolveScatterMetric(spec.label, spec.name, level); err != nil {
			return err
		}
	}

	return nil
}

func scatterMetricLevel(grain *string) (metric.MetricLevel, error) {
	switch value := ptrString(grain); value {
	case "", string(vizpkg.GrainFile):
		return metric.LevelFile, nil
	case string(vizpkg.GrainDirectory):
		return metric.LevelDirectory, nil
	default:
		return 0, eris.Errorf("unknown grain %q; must be \"file\" or \"directory\"", value)
	}
}

func resolveScatterMetric(
	label string,
	name metric.Name,
	level metric.MetricLevel,
) (provider.ResolvedMetric, error) {
	resolved, err := provider.ResolveName(name, level)
	if err != nil {
		return provider.ResolvedMetric{}, friendlyMetricError(label, name, err)
	}

	return resolved, nil
}

func validateScatterAxisMetric(label string, name *string, level metric.MetricLevel) error {
	metricName := metric.Name(ptrString(name))
	if metricName == "" {
		return eris.Errorf("%s metric is required", label)
	}

	_, err := resolveScatterMetric(label, metricName, level)

	return err
}

func validateScatterNumericMetric(label string, name metric.Name, level metric.MetricLevel) error {
	if name == "" {
		return eris.Errorf("%s metric is required", label)
	}

	resolved, err := resolveScatterMetric(label, name, level)
	if err != nil {
		return err
	}

	if resolved.ResultKind != metric.Quantity && resolved.ResultKind != metric.Measure {
		return eris.Errorf("%s metric must be numeric, got %q (kind: %d)", label, name, resolved.ResultKind)
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

//nolint:dupl // pipeline wiring is structurally similar across commands but not refactorable
func (c *ScatterCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	stagesFlags := stagesFlagsForCommand(flags, c.From, c.Until)

	common := &stages.CommonState{
		TargetPath:         c.TargetPath,
		Output:             c.Output,
		Flags:              stagesFlags,
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
	cfg.Scatter.OverrideGrain(c.Grain)
	cfg.OverrideLegendPosition(c.Legend)
	cfg.OverrideLegendOrientation(c.LegendOrientation)
}
