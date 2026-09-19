package alluvial

import (
	"time"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
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
	common.Requested = stages.CollectRequestedMetricNames(widthMetric)

	return nil
}

// AcquireData resolves and scans every caller-ordered reference independently.
func AcquireData(s *pipeline.State) {
	pipeline.ApplyFuncXYZ(s, acquireSnapshots)
	pipeline.ApplyFuncXY(s, BuildDataStage)
}

func acquireSnapshots(common *stages.CommonState, state *State, cfg *config.Alluvial) error {
	snapshots := make([]Snapshot, 0, len(cfg.References))
	for _, reference := range cfg.References {
		snapshotCommon, err := acquireSnapshot(common, reference)
		if err != nil {
			return eris.Wrapf(err, "failed to acquire alluvial reference %q", reference)
		}

		snapshots = append(snapshots, Snapshot{
			Reference: reference,
			Root:      snapshotCommon.Root,
		})
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
	snapshotFlags.HistoryRange.Until = reference
	snapshotFlags.HistoryRange.From = ""
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
		Metric: metricName,
		Expand: cfg.Expand,
	})
	if err != nil {
		return err
	}

	state.Data = data

	return nil
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
