package main

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/donuttree"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

type DonutTreeCmd struct {
	TargetPath string `arg:"" help:"Path to directory to scan."`
	Output     string `help:"Output image file path (png, jpg, jpeg, svg)." required:"true" short:"o"`
	From       string `help:"Git history lower bound: tag, commit ID, or date." name:"from" optional:""`
	Until      string `help:"Git history upper bound: tag, commit ID, or date." name:"until" optional:""`

	Size metric.Name `default:"" help:"Metric for folder sector size; run 'codeviz help metrics' for available metrics." short:"s"` //nolint:revive,nolintlint // kong struct tags require long lines

	Fill   config.MetricSpec `help:"Folder fill colour: metric[,palette] (defaults to size)." optional:"" short:"f"`
	Border config.MetricSpec `help:"Folder border colour: metric[,palette]." optional:"" short:"b"`

	Legend            string `default:"" enum:",top-left,top-center,top-right,center-right,bottom-right,bottom-center,bottom-left,center-left,none" help:"Legend position (default: bottom-right)." optional:""` //nolint:revive,nolintlint // kong struct tags require long lines
	LegendOrientation string `default:"" enum:",vertical,horizontal" help:"Legend orientation (auto-detected from position if omitted)." name:"legend-orientation" optional:""`                                  //nolint:revive,nolintlint // kong struct tags require long lines
	MaxLayers         int    `default:"0" help:"Maximum number of directory layers to display; 0 means unlimited." name:"max-layers"`                                                                            //nolint:revive,nolintlint // kong struct tags require long lines

	Width  int `help:"Image width in pixels."`
	Height int `help:"Image height in pixels."`

	Title      string `default:"" help:"Override title text on the generated image." optional:""`
	Footer     string `default:"" help:"Override footer text on the generated image." optional:""`
	HideFooter bool   `default:"false" help:"Suppress the attribution footer." name:"hide-footer" optional:""`

	Include            []filter.Rule `type:"filterrule" name:"include" help:"Include matching files (repeatable)." placeholder:"glob"`                 //nolint:revive,nolintlint // kong struct tags require long lines
	Exclude            []filter.Rule `type:"filterrule" name:"exclude" help:"Exclude matching files (repeatable)." placeholder:"glob"`                 //nolint:revive,nolintlint // kong struct tags require long lines
	IncludeBinaryFiles bool          `help:"Include binary files in the visualization (excluded by default)." name:"include-binary-files" optional:""` //nolint:revive,nolintlint // kong struct tags require long lines
}

func (c *DonutTreeCmd) Filters() []filter.Rule {
	return filter.Merge(c.Include, c.Exclude)
}

func (*DonutTreeCmd) Validate() error {
	return nil
}

func (*DonutTreeCmd) validateConfig(cfg *config.DonutTree) error {
	if err := validateNumericMetric("size", metric.Name(ptrString(cfg.Size))); err != nil {
		return err
	}

	if err := cfg.Fill.Validate("fill"); err != nil {
		return eris.Wrap(err, "invalid fill spec")
	}

	if err := cfg.Border.Validate("border"); err != nil {
		return eris.Wrap(err, "invalid border spec")
	}

	if cfg.MaxLayers != nil && *cfg.MaxLayers < 0 {
		return eris.Errorf("max layers must be >= 0")
	}

	return nil
}

func (c *DonutTreeCmd) mergeConfigAndValidate(flags *Flags) error {
	if err := flags.Config.TryAutoLoad(c.Output); err != nil {
		return eris.Wrap(err, "auto-config load failed")
	}

	c.applyOverrides(flags.Config)

	return c.validateConfig(flags.Config.DonutTree)
}

//nolint:dupl // each viz Run shares the same pipeline-construction boilerplate by design
func (c *DonutTreeCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	stagesFlags := stagesFlagsForCommand(flags, c.From, c.Until)

	common := &stages.CommonState{
		TargetPath:         c.TargetPath,
		Output:             c.Output,
		Flags:              stagesFlags,
		RootConfig:         flags.Config,
		VizName:            "donut-tree",
		CLIFilters:         c.Filters(),
		IncludeBinaryFiles: c.IncludeBinaryFiles,
	}
	cfg := flags.Config.DonutTree
	viz := &donuttree.State{}

	s := pipeline.NewState(common, cfg, viz)

	pipeline.ApplyFuncX(s, stages.ValidatePaths)
	pipeline.ApplyFuncX(s, stages.ExportConfig)
	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(s, donuttree.ResolveMetrics)

	donuttree.AcquireData(s)
	donuttree.RenderPipeline(s)

	return eris.Wrap(s.Err(), "donut-tree pipeline failed")
}

func (c *DonutTreeCmd) applyOverrides(cfg *config.Config) {
	cfg.OverrideWidth(c.Width)
	cfg.OverrideHeight(c.Height)
	cfg.OverrideTitleText(c.Title)
	cfg.OverrideFooterText(c.Footer)
	cfg.OverrideHideFooter(c.HideFooter)

	if cfg.DonutTree == nil {
		cfg.DonutTree = &config.DonutTree{}
	}

	cfg.DonutTree.OverrideSize(string(c.Size))
	cfg.DonutTree.OverrideFill(c.Fill)
	cfg.DonutTree.OverrideBorder(c.Border)
	cfg.DonutTree.OverrideMaxLayers(c.MaxLayers)
	cfg.OverrideLegendPosition(c.Legend)
	cfg.OverrideLegendOrientation(c.LegendOrientation)
}
