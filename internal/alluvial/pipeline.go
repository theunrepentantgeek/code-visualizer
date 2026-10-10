package alluvial

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/progress"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	gitprovider "github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
	"github.com/theunrepentantgeek/code-visualizer/internal/source"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics prepares the configured metric for scanning and aggregation.
func ResolveMetrics(common *stages.CommonState, state *State, cfg *config.Alluvial) error {
	if cfg == nil || cfg.Metric == nil {
		return eris.New("alluvial width metric is required")
	}

	widthMetric, err := resolveDirectoryMetric(metric.Name(*cfg.Metric))
	if err != nil {
		return eris.Wrap(err, "invalid alluvial width metric")
	}

	state.WidthMetric = widthMetric

	labelMetrics, err := resolveLabelMetrics(cfg.Labels)
	if err != nil {
		return err
	}

	state.LabelMetrics = labelMetrics

	fill, err := resolveBandFill(cfg.Fill, widthMetric)
	if err != nil {
		return err
	}

	state.Fill = fill

	requested := make([]metric.Name, 0, 2+len(labelMetrics))
	requested = append(requested, widthMetric, fill.Encoding.Metric)
	requested = append(requested, labelMetrics...)
	common.Requested = stages.CollectRequestedMetricNames(requested...)

	return nil
}

func resolveLabelMetrics(names []metric.Name) ([]metric.Name, error) {
	resolved := make([]metric.Name, 0, len(names))
	for _, name := range names {
		resolvedName, err := resolveDirectoryMetric(name)
		if err != nil {
			return nil, eris.Wrap(err, "invalid alluvial label metric")
		}

		resolved = append(resolved, resolvedName)
	}

	return resolved, nil
}

func resolveBandFill(fill *config.MetricSpec, widthMetric metric.Name) (BandFill, error) {
	if fill == nil || fill.Metric == "" {
		return BandFill{
			Encoding: stages.ResolveColourEncodingForMetric(fill, widthMetric),
			Label:    widthMetric,
		}, nil
	}

	expression, err := metric.ParseExpression(string(fill.Metric))
	if err != nil {
		return BandFill{}, eris.Wrap(err, "parse alluvial fill metric")
	}

	fillMetric, err := resolveDirectoryExpression(expression)
	if err != nil {
		return BandFill{}, eris.Wrap(err, "invalid alluvial fill metric")
	}

	return BandFill{
		Encoding: stages.ResolveColourEncodingForMetric(fill, fillMetric),
		Label:    fill.Metric,
		Explicit: true,
		Temporal: expression.Temporal,
	}, nil
}

func FinalizeData(s *pipeline.State) {
	s.ApplyFuncXY(BuildDataStage).
		ApplyFuncXY(BuildLegendStage)
}

// RenderPipeline lays out the acquired snapshot data and writes the shared
// canvas output.
func RenderPipeline(s *pipeline.State) {
	RenderVisualization(s)
	WriteOutput(s)
}

func RenderVisualization(s *pipeline.State) {
	s.ApplyFuncX(stages.ResolveDimensions).
		ApplyFuncX(stages.InitDrawingBounds).
		ApplyFuncX(stages.ReserveTitleBounds).
		ApplyFuncX(stages.ReserveFooterBounds).
		ApplyFuncXY(LayoutStage).
		ApplyFuncXY(RenderStage).
		ApplyFuncX(stages.ApplyTitle).
		ApplyFuncX(stages.ApplyFooter)
}

func WriteOutput(s *pipeline.State) {
	s.ApplyFuncX(stages.WriteCanvas)
}

// LayoutStage reserves legend space, assigns metric-proportional vertical
// bands inside the drawable area, and applies the resulting offset.
func LayoutStage(common *stages.CommonState, state *State) error {
	bounds := common.DrawingBounds
	reservation := legend.ReserveLayout(state.Legend, common.Width, int(bounds.Height()))
	layout := LayoutData(state.Data, reservation.Width, reservation.Height)
	offset := reservation.Offset.Add(geometry.NewVector(0, bounds.Min.Y))
	offsetLayout(&layout, offset)
	state.Layout = layout

	return nil
}

// RenderStage creates the shared-canvas shapes from the positioned layout.
func RenderStage(common *stages.CommonState, state *State) error {
	common.Canvas = RenderToCanvas(
		state.Layout,
		common.Width,
		common.Height,
		state.Fill.Ink,
		state.Fill.LabelMetric(),
	)
	legend.RenderInto(common.Canvas, state.Legend)

	return nil
}

func offsetLayout(layout *Layout, offset geometry.Vector) {
	layout.Top += offset.Y

	layout.Bottom += offset.Y
	for columnIndex := range layout.Columns {
		layout.Columns[columnIndex].X += offset.X
		for bandIndex := range layout.Columns[columnIndex].Bands {
			layout.Columns[columnIndex].Bands[bandIndex].Top += offset.Y
			layout.Columns[columnIndex].Bands[bandIndex].Bottom += offset.Y
		}
	}

	for flowIndex := range layout.Flows {
		layout.Flows[flowIndex].FromX += offset.X
		layout.Flows[flowIndex].ToX += offset.X
		layout.Flows[flowIndex].FromTop += offset.Y
		layout.Flows[flowIndex].FromBottom += offset.Y
		layout.Flows[flowIndex].ToTop += offset.Y
		layout.Flows[flowIndex].ToBottom += offset.Y
	}
}

func (p *AcquisitionPlan) PrepareReferences(
	ctx context.Context,
	sink progress.Sink,
	common *stages.CommonState,
	state *State,
	cfg *config.Alluvial,
) error {
	snapshotStates, err := prepareSnapshots(ctx, sink, common, cfg.References)
	if err != nil {
		return err
	}

	err = shareCommitHistoryPaths(common, snapshotStates)
	if err != nil {
		return err
	}

	p.snapshotStates = snapshotStates

	p.expansions = append([]string(nil), cfg.Expand...)
	p.expressions = append([]provider.ResolvedMetric(nil), common.Requested.Expressions...)
	state.Snapshots = nil

	return nil
}

func prepareSnapshots(
	ctx context.Context,
	sink progress.Sink,
	common *stages.CommonState,
	references []string,
) ([]*stages.CommonState, error) {
	snapshotStates := make([]*stages.CommonState, 0, len(references))
	for _, reference := range references {
		snapshotCommon, err := prepareSnapshot(ctx, sink, common, reference)
		if err != nil {
			return nil, eris.Wrapf(err, "failed to acquire alluvial reference %q", reference)
		}

		snapshotStates = append(snapshotStates, snapshotCommon)
	}

	return snapshotStates, nil
}

func shareCommitHistoryPaths(
	common *stages.CommonState,
	snapshotStates []*stages.CommonState,
) error {
	if !common.Requested.HasCommitExpressions() {
		return nil
	}

	if err := stages.ShareGitHistoryPaths(snapshotStates); err != nil {
		return eris.Wrap(err, "prepare shared alluvial history")
	}

	return nil
}

func (p *AcquisitionPlan) AcquireReference(
	ctx context.Context,
	sink progress.Sink,
	state *State,
	reference string,
	index int,
) error {
	if index < 0 || index >= len(p.snapshotStates) {
		return eris.Errorf("alluvial reference index %d is out of range", index)
	}

	snapshotCommon := p.snapshotStates[index]
	if err := finishSnapshot(ctx, sink, snapshotCommon); err != nil {
		return eris.Wrapf(err, "failed to acquire alluvial reference %q", reference)
	}

	bands, err := evaluateSnapshotBands(
		snapshotCommon.Root,
		p.expansions,
		p.expressions,
	)
	if err != nil {
		return eris.Wrapf(err, "failed to evaluate alluvial bands for reference %q", reference)
	}

	state.Snapshots = append(state.Snapshots, Snapshot{
		Reference: SnapshotReference(reference),
		Bands:     bands,
	})

	if index == 0 {
		if err := stages.ExportData(snapshotCommon); err != nil {
			return eris.Wrap(err, "export alluvial snapshot data")
		}
	}

	return nil
}

func evaluateSnapshotBands(
	root *model.Directory,
	expansions []string,
	expressions []provider.ResolvedMetric,
) (map[string]*model.MetricContainer, error) {
	selections := model.PartitionDirectories(root, expansions)
	bands := make(map[string]*model.MetricContainer, len(selections))

	for directoryPath, selection := range selections {
		values, err := stages.EvaluateAggregations(selection, expressions)
		if err != nil {
			return nil, eris.Wrapf(err, "evaluate directory band %q", directoryPath)
		}

		bands[directoryPath] = values
	}

	return bands, nil
}

func prepareSnapshot(
	ctx context.Context,
	sink progress.Sink,
	common *stages.CommonState,
	reference string,
) (*stages.CommonState, error) {
	if common.Flags == nil {
		return nil, eris.New("alluvial snapshot acquisition requires pipeline flags")
	}

	snapshotCommon := *common
	snapshotFlags := *common.Flags
	snapshotFlags.HistoryRange.Until = SnapshotReference(reference)
	snapshotFlags.HistoryRange.From = ""
	snapshotFlags.ChangedOnly = false
	snapshotCommon.Flags = &snapshotFlags
	snapshotCommon.Root = nil
	snapshotCommon.Source = source.Tree{}
	snapshotCommon.Snapshot = nil
	snapshotCommon.ReferenceNow = time.Time{}

	for _, stage := range []func(*stages.CommonState) error{
		func(state *stages.CommonState) error {
			return stages.ScanFilesystem(ctx, state, sink)
		},
		stages.CheckGitRequirement,
	} {
		if err := stage(&snapshotCommon); err != nil {
			recovered, recoveryErr := recoverEmptySnapshot(&snapshotCommon, err)
			if recoveryErr != nil {
				return nil, recoveryErr
			}

			if recovered {
				continue
			}

			return nil, err
		}
	}

	return &snapshotCommon, nil
}

func recoverEmptySnapshot(common *stages.CommonState, scanErr error) (bool, error) {
	if !errors.Is(scanErr, scan.ErrNoFiles) && !gitprovider.IsSnapshotTargetMissing(scanErr) {
		return false, nil
	}

	emptyRoot, err := emptySnapshotRoot(common)
	if err != nil {
		return false, eris.Wrap(err, "prepare empty alluvial snapshot")
	}

	common.Root = emptyRoot

	return true, nil
}

func emptySnapshotRoot(common *stages.CommonState) (*model.Directory, error) {
	absolute, err := filepath.Abs(common.TargetPath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to resolve empty snapshot target")
	}

	repoPath, err := filepath.Rel(common.RepoRoot, absolute)
	if err != nil {
		return nil, eris.Wrap(err, "failed to resolve empty snapshot repository path")
	}

	return &model.Directory{
		Path:     absolute,
		RepoPath: filepath.ToSlash(repoPath),
		RepoRoot: common.RepoRoot,
		Name:     filepath.Base(absolute),
	}, nil
}

func finishSnapshot(
	ctx context.Context,
	sink progress.Sink,
	snapshotCommon *stages.CommonState,
) error {
	if model.CountFiles(snapshotCommon.Root) == 0 {
		return nil
	}

	for _, stage := range []func(*stages.CommonState) error{
		func(state *stages.CommonState) error {
			return stages.LoadCommitMetrics(ctx, state, sink)
		},
		func(state *stages.CommonState) error {
			return stages.RunProviders(ctx, state, sink)
		},
		stages.PopulateDeclarations,
		stages.RunAggregations,
		stages.FilterBinaryFiles,
	} {
		if err := stage(snapshotCommon); err != nil {
			return err
		}
	}

	return nil
}

// SnapshotReference normalizes an empty alluvial reference to the current Git
// commit while preserving explicit tag, SHA, and date references.
func SnapshotReference(reference string) string {
	if strings.TrimSpace(reference) == "" {
		return "HEAD"
	}

	return reference
}

// BuildDataStage converts acquired snapshots into the renderer-independent
// columns and transitions consumed by U3's layout and rendering stages.
func BuildDataStage(state *State, cfg *config.Alluvial) error {
	if cfg == nil || cfg.Metric == nil {
		return eris.New("alluvial width metric is required")
	}

	metricName := state.WidthMetric
	if metricName == "" {
		metricName = metric.Name(*cfg.Metric)
	}

	data, err := BuildData(state.Snapshots, Options{
		Metric:        metricName,
		LabelMetrics:  state.LabelMetrics,
		FillMetric:    state.Fill.Encoding.Metric,
		FillTemporal:  state.Fill.Temporal,
		ConstantBands: cfg.ConstantBandsMode(),
	})
	if err != nil {
		return err
	}

	state.Data = data

	return nil
}

// BuildLegendStage creates the metric-driven band ink and its colour key.
func BuildLegendStage(common *stages.CommonState, state *State) error {
	state.Fill.ResolveInk(state.Data.FillValuesForInk())

	rootConfig := common.RootConfig
	if rootConfig == nil {
		rootConfig = config.New()
	}

	position, orientation := legend.ResolveOptions(rootConfig.LegendPositionStr(), rootConfig.LegendOrientationStr())

	state.Legend = legend.Builder{
		Position:    position,
		Orientation: orientation,
		FillInk:     state.Fill.Ink,
		FillMetric:  state.Fill.Label,
		SizeMetric:  state.WidthMetric,
	}.Build()
	if state.Legend != nil {
		lines := []string{"Directory", string(state.WidthMetric)}
		if state.Fill.Explicit {
			lines = append(lines, string(state.Fill.Label))
		}

		for _, name := range state.LabelMetrics {
			lines = append(lines, string(name))
		}

		state.Legend.LabelSample = legend.LabelSample{
			Shape: legend.LabelSampleSquare,
			Lines: lines,
		}
	}

	return nil
}

func resolveDirectoryMetric(name metric.Name) (metric.Name, error) {
	expression, err := metric.ParseExpression(string(name))
	if err != nil {
		return "", eris.Wrap(err, "parse metric expression")
	}

	if !expression.Temporal.IsZero() {
		return "", eris.Errorf("temporal modifier %q is not supported for alluvial width", expression.Temporal)
	}

	return resolveDirectoryExpression(expression)
}

func resolveDirectoryExpression(expression metric.MetricExpression) (metric.Name, error) {
	descriptor, ok := provider.GetBase(expression.Base)
	if !ok {
		return "", eris.Errorf("unknown base metric %q", expression.Base)
	}

	if expression.Aggregation.IsZero() {
		aggregation, ok := descriptor.Kind.DefaultAggregation()
		if !ok {
			return "", eris.Errorf("unsupported metric kind %d", descriptor.Kind)
		}

		expression.Aggregation = aggregation
	}

	resolved, err := provider.ResolveExpression(expression, metric.LevelDirectory)
	if err != nil {
		return "", eris.Wrap(err, "resolve metric expression")
	}

	return resolved.ResultName, nil
}
