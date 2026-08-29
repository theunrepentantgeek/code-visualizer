package main

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/radialtree"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

type RadialCmd struct {
	TargetPath string `arg:"" help:"Path to directory to scan."`
	Output     string `help:"Output image file path (png, jpg, jpeg, svg)." required:"true" short:"o"`

	FileDiscSize metric.Name `default:"" help:"Metric for file disc size; run 'codeviz help metrics' for available metrics." name:"file-disc-size" short:"d"` //nolint:revive,nolintlint // kong struct tags require long lines

	FileFill   config.MetricSpec `help:"File fill colour: metric[,palette] (e.g. file-type,categorization)." name:"file-fill" optional:"" short:"f"` //nolint:revive,nolintlint // kong struct tags require long lines
	FileBorder config.MetricSpec `help:"File border colour: metric[,palette] (e.g. file-lines,foliage)." name:"file-border" optional:"" short:"b"`   //nolint:revive,nolintlint // kong struct tags require long lines

	DirectoryDiscSize metric.Name       `default:"" help:"Metric for directory disc size; run 'codeviz help metrics' for available metrics." name:"directory-disc-size"` //nolint:revive,nolintlint // kong struct tags require long lines
	DirectoryFill     config.MetricSpec `help:"Directory fill colour: metric[,palette] (defaults to aggregated file fill)." name:"directory-fill" optional:""`           //nolint:revive,nolintlint // kong struct tags require long lines
	DirectoryBorder   config.MetricSpec `help:"Directory border colour: metric[,palette] (defaults to aggregated file border)." name:"directory-border" optional:""`     //nolint:revive,nolintlint // kong struct tags require long lines

	Labels string `enum:",all,folders,none" default:"" help:"Labels to display: all, folders, or none."`

	Grain     string `enum:",file,directory" default:"" help:"Granularity of nodes shown: file (default) or directory."`      //nolint:revive,nolintlint // kong struct tags require long lines
	MaxLayers int    `default:"0" help:"Maximum number of directory layers to display; 0 means unlimited." name:"max-layers"` //nolint:revive,nolintlint // kong struct tags require long lines

	Legend            string `default:"" enum:",top-left,top-center,top-right,center-right,bottom-right,bottom-center,bottom-left,center-left,none" help:"Legend position (default: bottom-right)." optional:""` //nolint:revive,nolintlint // kong struct tags require long lines
	LegendOrientation string `default:"" enum:",vertical,horizontal" help:"Legend orientation (auto-detected from position if omitted)." name:"legend-orientation" optional:""`                                  //nolint:revive,nolintlint // kong struct tags require long lines

	Width  int `default:"1920" help:"Image width in pixels."`
	Height int `default:"1920" help:"Image height in pixels."`

	Title      string `default:"" help:"Override title text on the generated image." optional:""`
	Footer     string `default:"" help:"Override footer text on the generated image." optional:""`
	HideFooter bool   `default:"false" help:"Suppress the attribution footer." name:"hide-footer" optional:""`

	Include            []filter.Rule `type:"filterrule" name:"include" help:"Include matching files (repeatable)." placeholder:"glob"`                 //nolint:revive,nolintlint // kong struct tags require long lines
	Exclude            []filter.Rule `type:"filterrule" name:"exclude" help:"Exclude matching files (repeatable)." placeholder:"glob"`                 //nolint:revive,nolintlint // kong struct tags require long lines
	IncludeBinaryFiles bool          `help:"Include binary files in the visualization (excluded by default)." name:"include-binary-files" optional:""` //nolint:revive // kong struct tags require long lines
}

func (c *RadialCmd) Filters() []filter.Rule {
	return filter.Merge(c.Include, c.Exclude)
}

func (*RadialCmd) Validate() error {
	return nil
}

// validateConfig checks the effective configuration after all sources have been
// merged. Called from mergeConfigAndValidate() after TryAutoLoad + applyOverrides.
func (*RadialCmd) validateConfig(cfg *config.Radial) error {
	if err := validateNumericMetric("file-disc-size", metric.Name(ptrString(cfg.FileDiscSize))); err != nil {
		return err
	}

	if cfg.DirectoryDiscSize != nil {
		if err := validateNumericMetric(
			"directory-disc-size",
			metric.Name(ptrString(cfg.DirectoryDiscSize)),
		); err != nil {
			return err
		}
	}

	if err := cfg.FileFill.Validate("file fill"); err != nil {
		return eris.Wrap(err, "invalid file fill spec")
	}

	if err := cfg.FileBorder.Validate("file border"); err != nil {
		return eris.Wrap(err, "invalid file border spec")
	}

	if err := cfg.DirectoryFill.Validate("directory fill"); err != nil {
		return eris.Wrap(err, "invalid directory fill spec")
	}

	if err := cfg.DirectoryBorder.Validate("directory border"); err != nil {
		return eris.Wrap(err, "invalid directory border spec")
	}

	if cfg.MaxLayers != nil && *cfg.MaxLayers < 0 {
		return eris.Errorf("max layers must be >= 0")
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

//nolint:dupl // each viz Run shares the same pipeline-construction boilerplate by design
func (c *RadialCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	common := &stages.CommonState{
		TargetPath:         c.TargetPath,
		Output:             c.Output,
		Flags:              toStagesFlags(flags),
		RootConfig:         flags.Config,
		VizName:            "radial-tree",
		CLIFilters:         c.Filters(),
		IncludeBinaryFiles: c.IncludeBinaryFiles,
	}
	cfg := flags.Config.Radial
	viz := &radialtree.State{}

	s := pipeline.NewState(common, cfg, viz)

	pipeline.ApplyFuncX(s, stages.ValidatePaths)
	pipeline.ApplyFuncX(s, stages.ExportConfig)
	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(s, radialtree.ResolveMetrics)

	radialtree.AcquireData(s)
	radialtree.RenderPipeline(s)

	return eris.Wrap(s.Err(), "radialtree pipeline failed")
}

// applyOverrides writes non-zero CLI flag values on top of the config layer.
// Zero-valued CLI fields are transparent — the config value passes through unchanged.
func (c *RadialCmd) applyOverrides(cfg *config.Config) {
	cfg.OverrideWidth(c.Width)
	cfg.OverrideHeight(c.Height)
	cfg.OverrideTitleText(c.Title)
	cfg.OverrideFooterText(c.Footer)
	cfg.OverrideHideFooter(c.HideFooter)

	if cfg.Radial == nil {
		cfg.Radial = &config.Radial{}
	}

	cfg.Radial.OverrideFileDiscSize(string(c.FileDiscSize))
	cfg.Radial.OverrideFileFill(c.FileFill)
	cfg.Radial.OverrideFileBorder(c.FileBorder)
	cfg.Radial.OverrideDirectoryDiscSize(string(c.DirectoryDiscSize))
	cfg.Radial.OverrideDirectoryFill(c.DirectoryFill)
	cfg.Radial.OverrideDirectoryBorder(c.DirectoryBorder)
	cfg.Radial.OverrideLabels(c.Labels)
	cfg.Radial.OverrideGrain(c.Grain)
	cfg.Radial.OverrideMaxLayers(c.MaxLayers)
	cfg.OverrideLegendPosition(c.Legend)
	cfg.OverrideLegendOrientation(c.LegendOrientation)
}
