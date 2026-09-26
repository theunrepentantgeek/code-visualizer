package main

import (
	"context"
	"path"
	"strings"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/alluvial"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/progress"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

type AlluvialCmd struct {
	TargetPath string `arg:"" help:"Path to directory to scan."`
	Output     string `help:"Output image file path (png, jpg, jpeg, svg)." required:"true" short:"o"`

	References []string          `help:"Ordered revision reference (repeatable)." name:"reference"`
	Metric     metric.Name       `default:"" help:"Metric for directory flow width; run 'codeviz help metrics' for available metrics." short:"m"`              //nolint:revive,nolintlint // kong struct tags require long lines
	Fill       config.MetricSpec `help:"Band colour: metric[,palette]; append .delta to colour by change from first to last reference." optional:"" short:"f"` //nolint:revive,nolintlint // kong struct tags require long lines

	Expand []string `help:"Directory whose direct children should be shown (repeatable)." placeholder:"path"`

	Width  int `default:"1920" help:"Image width in pixels."`
	Height int `default:"1080" help:"Image height in pixels."`

	Title      string `default:"" help:"Override title text on the generated image." optional:""`
	Footer     string `default:"" help:"Override footer text on the generated image." optional:""`
	HideFooter bool   `default:"false" help:"Suppress the attribution footer." name:"hide-footer" optional:""`

	Include            []filter.Rule `type:"filterrule" name:"include" help:"Include matching files (repeatable)." placeholder:"glob"`                 //nolint:revive,nolintlint // kong struct tags require long lines
	Exclude            []filter.Rule `type:"filterrule" name:"exclude" help:"Exclude matching files (repeatable)." placeholder:"glob"`                 //nolint:revive,nolintlint // kong struct tags require long lines
	IncludeBinaryFiles bool          `help:"Include binary files in the visualization (excluded by default)." name:"include-binary-files" optional:""` //nolint:revive,nolintlint // kong struct tags require long lines
}

func (c *AlluvialCmd) Filters() []filter.Rule {
	return filter.Merge(c.Include, c.Exclude)
}

func (*AlluvialCmd) validateConfig(cfg *config.Alluvial) error {
	if len(cfg.References) < 2 {
		return eris.New("alluvial requires at least two references")
	}

	references := make(map[string]struct{}, len(cfg.References))
	for _, reference := range cfg.References {
		reference = alluvial.SnapshotReference(reference)
		if _, exists := references[reference]; exists {
			return eris.Errorf("alluvial references must be unique: duplicate reference %q", reference)
		}

		references[reference] = struct{}{}
	}

	metricName := metric.Name(ptrString(cfg.Metric))
	if metricName == "" {
		return eris.New("metric is required")
	}

	if err := validateNumericMetric("metric", metricName); err != nil {
		return err
	}

	if err := validateAlluvialFill(cfg.Fill, metricName); err != nil {
		return err
	}

	for _, expansion := range cfg.Expand {
		if err := validateExpansionPath(expansion); err != nil {
			return err
		}
	}

	return nil
}

func validateAlluvialFill(fillSpec *config.MetricSpec, defaultMetric metric.Name) error {
	if fillSpec == nil {
		return nil
	}

	fillMetric := fillSpec.Metric
	if fillMetric == "" {
		fillMetric = defaultMetric
	}

	fillMetric, _ = alluvial.ParseFillMetric(fillMetric)
	if err := validateNumericMetric("fill", fillMetric); err != nil {
		return err
	}

	fill := *fillSpec

	fill.Metric = fillMetric
	if err := fill.Validate("fill"); err != nil {
		return eris.Wrap(err, "invalid fill spec")
	}

	return nil
}

func validateExpansionPath(expansion string) error {
	cleaned := path.Clean(expansion)
	if strings.TrimSpace(expansion) == "" || path.IsAbs(expansion) ||
		cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return eris.Errorf("invalid expansion path %q: must be repository-relative", expansion)
	}

	return nil
}

func (c *AlluvialCmd) mergeConfigAndValidate(flags *Flags) error {
	if err := flags.Config.TryAutoLoad(c.Output); err != nil {
		return eris.Wrap(err, "auto-config load failed")
	}

	c.applyOverrides(flags.Config)

	return c.validateConfig(flags.Config.Alluvial)
}

// Run resolves each reference snapshot, lays out aligned flows, and writes the
// requested shared-canvas image.
func (c *AlluvialCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	common := &stages.CommonState{
		TargetPath:         c.TargetPath,
		Output:             c.Output,
		Flags:              stagesFlagsForCommand(flags, "", flags.Config.Alluvial.References[0]),
		RootConfig:         flags.Config,
		VizName:            "alluvial",
		CLIFilters:         c.Filters(),
		IncludeBinaryFiles: c.IncludeBinaryFiles,
	}
	viz := &alluvial.State{}
	cfg := flags.Config.Alluvial
	s := pipeline.NewState(common, cfg, viz)

	phases := buildAlluvialPhases(common, viz, cfg)

	return eris.Wrap(runCommandWorkflow(flags, s, "Alluvial", phases), "alluvial pipeline failed")
}

func buildAlluvialPhases(
	common *stages.CommonState,
	viz *alluvial.State,
	cfg *config.Alluvial,
) []workflowPhase {
	phases := make([]workflowPhase, 0, len(cfg.References)+3)

	phases = append(phases, workflowPhase{
		Name: phasePreparing,
		Kind: progress.StageSummary,
		Run: func(s *pipeline.State) {
			pipeline.ApplyFuncX(s, stages.ValidatePaths)
			pipeline.ApplyFuncX(s, stages.ExportConfig)
			pipeline.ApplyFuncX(s, stages.BuildFilterRules)
			pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
			pipeline.ApplyFuncXYZ(s, alluvial.ResolveMetrics)
			pipeline.ApplyFuncXYZ(s, func(
				ctx context.Context,
				sink progress.Sink,
				common *stages.CommonState,
			) error {
				return alluvial.PrepareReferences(ctx, sink, common, viz, cfg)
			})

			work := stages.AcquisitionWork(common)
			for index := range cfg.References {
				phases[index+1].Work = work
			}
		},
	})
	for index, reference := range cfg.References {
		phases = append(phases, workflowPhase{
			Name: "Loading " + alluvial.SnapshotReference(reference),
			Kind: progress.StageLive,
			Run: func(s *pipeline.State) {
				pipeline.ApplyFuncXY(s, func(ctx context.Context, sink progress.Sink) error {
					return alluvial.AcquireReference(ctx, sink, viz, reference, index)
				})
			},
		})
	}

	phases = append(
		phases,
		workflowPhase{Name: phaseRendering, Kind: progress.StageSummary, Run: func(s *pipeline.State) {
			alluvial.FinalizeData(s)
			alluvial.RenderVisualization(s)
		}},
		workflowPhase{Name: phaseWriting, Kind: progress.StageSummary, Run: alluvial.WriteOutput},
	)

	return phases
}

func (c *AlluvialCmd) applyOverrides(cfg *config.Config) {
	cfg.OverrideWidth(c.Width)
	cfg.OverrideHeight(c.Height)
	cfg.OverrideTitleText(c.Title)
	cfg.OverrideFooterText(c.Footer)
	cfg.OverrideHideFooter(c.HideFooter)

	if cfg.Alluvial == nil {
		cfg.Alluvial = &config.Alluvial{}
	}

	cfg.Alluvial.OverrideReferences(c.References)
	cfg.Alluvial.OverrideMetric(string(c.Metric))
	cfg.Alluvial.OverrideFill(c.Fill)
	cfg.Alluvial.OverrideExpand(c.Expand)
}
