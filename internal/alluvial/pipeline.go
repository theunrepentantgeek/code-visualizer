package alluvial

import (
	"strings"
	"time"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
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
	fillMetric := widthMetric
	fillLabel := widthMetric
	fillDelta := false

	if cfg.Fill != nil && cfg.Fill.Metric != "" {
		state.FillSpecified = true
		fillLabel = cfg.Fill.Metric
		fillMetric, fillDelta = ParseFillMetric(cfg.Fill.Metric)

		fillMetric, err = resolveDirectoryMetric(fillMetric)
		if err != nil {
			return eris.Wrap(err, "invalid alluvial fill metric")
		}
	}

	state.Fill = stages.ResolveColourEncodingForMetric(cfg.Fill, fillMetric)
	state.FillLabel = fillLabel
	state.FillDelta = fillDelta
	common.Requested = stages.CollectRequestedMetricNames(widthMetric, fillMetric)

	return nil
}

// AcquireData resolves and scans every caller-ordered reference independently.
func AcquireData(s *pipeline.State) {
	pipeline.ApplyFuncXYZ(s, acquireSnapshots)
	pipeline.ApplyFuncXY(s, BuildDataStage)
	pipeline.ApplyFuncXY(s, BuildLegendStage)
}

// RenderPipeline lays out the acquired snapshot data and writes the shared
// canvas output.
func RenderPipeline(s *pipeline.State) {
	pipeline.ApplyFuncX(s, stages.ResolveDimensions)
	pipeline.ApplyFuncX(s, stages.InitDrawingBounds)
	pipeline.ApplyFuncX(s, stages.ReserveTitleBounds)
	pipeline.ApplyFuncX(s, stages.ReserveFooterBounds)
	pipeline.ApplyFuncXY(s, LayoutStage)
	pipeline.ApplyFuncXY(s, RenderStage)
	pipeline.ApplyFuncX(s, stages.ApplyTitle)
	pipeline.ApplyFuncX(s, stages.ApplyFooter)
	pipeline.ApplyFuncX(s, stages.WriteCanvas)
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
	labelFillMetric := metric.Name("")
	if state.FillSpecified {
		labelFillMetric = state.FillLabel
	}

	common.Canvas = RenderToCanvas(
		state.Layout,
		common.Width,
		common.Height,
		state.FillInk,
		labelFillMetric,
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

func acquireSnapshots(common *stages.CommonState, state *State, cfg *config.Alluvial) error {
	snapshots := make([]Snapshot, 0, len(cfg.References))
	for index, reference := range cfg.References {
		snapshotCommon, err := acquireSnapshot(common, reference)
		if err != nil {
			return eris.Wrapf(err, "failed to acquire alluvial reference %q", reference)
		}

		snapshots = append(snapshots, Snapshot{
			Reference: SnapshotReference(reference),
			Root:      snapshotCommon.Root,
		})

		if index == 0 {
			if err := stages.ExportData(snapshotCommon); err != nil {
				return eris.Wrap(err, "export alluvial snapshot data")
			}
		}
	}

	state.Snapshots = snapshots

	return nil
}

func acquireSnapshot(common *stages.CommonState, reference string) (*stages.CommonState, error) {
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
		stages.ScanFilesystem,
		stages.CheckGitRequirement,
		stages.RunProviders,
		stages.PopulateDeclarations,
		stages.RunAggregations,
		stages.FilterBinaryFiles,
	} {
		if err := stage(&snapshotCommon); err != nil {
			return nil, err
		}
	}

	return &snapshotCommon, nil
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
		Metric:     metricName,
		FillMetric: state.Fill.Metric,
		FillDelta:  state.FillDelta,
		Expand:     cfg.Expand,
	})
	if err != nil {
		return err
	}

	state.Data = data

	return nil
}

// BuildLegendStage creates the metric-driven band ink and its colour key.
func BuildLegendStage(common *stages.CommonState, state *State) error {
	values := make([]float64, 0, len(state.Data.FillValues))
	for _, value := range state.Data.FillValues {
		values = append(values, value)
		if state.FillDelta {
			values = append(values, -value)
		}
	}

	if state.FillDelta {
		values = append(values, 0)
	}

	state.FillInk = inks.NumericInk(state.FillLabel, values, palette.GetPalette(state.Fill.Palette))

	rootConfig := common.RootConfig
	if rootConfig == nil {
		rootConfig = config.New()
	}

	position, orientation := legend.ResolveOptions(rootConfig.LegendPositionStr(), rootConfig.LegendOrientationStr())

	state.Legend = legend.Builder{
		Position:    position,
		Orientation: orientation,
		FillInk:     state.FillInk,
		FillMetric:  state.FillLabel,
		SizeMetric:  state.WidthMetric,
	}.Build()
	if state.Legend != nil {
		lines := []string{"Directory", string(state.WidthMetric)}
		if state.FillSpecified {
			lines = append(lines, string(state.FillLabel))
		}

		state.Legend.LabelSample = legend.LabelSample{
			Shape: legend.LabelSampleSquare,
			Lines: lines,
		}
	}

	return nil
}

// ParseFillMetric separates the alluvial-only delta modifier from its metric.
func ParseFillMetric(name metric.Name) (metric.Name, bool) {
	const suffix = ".delta"

	raw := string(name)
	if !strings.HasSuffix(raw, suffix) {
		return name, false
	}

	return metric.Name(strings.TrimSuffix(raw, suffix)), true
}

func resolveDirectoryMetric(name metric.Name) (metric.Name, error) {
	expression, err := metric.ParseExpression(string(name))
	if err != nil {
		return "", eris.Wrap(err, "parse metric expression")
	}

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

	if _, err := provider.ResolveExpression(expression, metric.LevelDirectory); err != nil {
		return "", eris.Wrap(err, "resolve metric expression")
	}

	return expression.ResultName(), nil
}
