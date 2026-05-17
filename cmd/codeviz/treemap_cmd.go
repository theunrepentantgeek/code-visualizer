package main

import (
	"strings"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
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

// mergeConfigAndValidate loads the config file, merges CLI overrides on top,
// and validates the effective configuration. Called at the start of Run().
func (c *TreemapCmd) mergeConfigAndValidate(flags *Flags) error {
	if err := flags.Config.TryAutoLoad(c.Output); err != nil {
		return eris.Wrap(err, "auto-config load failed")
	}

	c.applyOverrides(flags.Config)

	return c.validateConfig(flags.Config.Treemap)
}

func (c *TreemapCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	state := &treemap.State{
		CommonState: stages.CommonState{
			TargetPath: c.TargetPath,
			Output:     c.Output,
			Flags:      toStagesFlags(flags),
			RootConfig: flags.Config,
			CLIFilters: c.Filter,
		},
		Config:             flags.Config.Treemap,
		IncludeBinaryFiles: c.IncludeBinaryFiles,
	}

	_, err := pipeline.Run[*treemap.State](
		state,
		stages.ValidatePaths,
		stages.ExportConfig,
		stages.BuildFilterRules,
		treemap.ResolveMetrics,
		stages.ScanFilesystem,
		stages.CheckGitRequirement,
		stages.RunProviders,
		stages.FilterBinaryFiles,
		stages.ExportData,
		stages.ResolveDimensions,
		treemap.BuildInksStage,
		treemap.BuildLegendStage,
		treemap.LayoutStage,
		treemap.RenderStage,
		stages.WriteCanvas,
		treemap.LogResult,
	)

	return eris.Wrap(err, "treemap pipeline failed")
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

// ptrInt safely dereferences a *int, returning a default 1920 if nil.
func ptrInt(p *int) int {
	if p == nil {
		return 1920
	}

	return *p
}
