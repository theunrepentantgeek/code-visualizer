package spiral

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves size, fill, border, resolution, and label settings
// from the spiral config and populates Common().Requested.
func ResolveMetrics(s *State) error {
	cfg := s.Config

	s.Size = metric.Name(stages.PtrString(cfg.Size))
	s.FillMetric = cfg.Fill.MetricName()
	s.FillPalette = stages.ResolveFillPalette(cfg.Fill, s.FillMetric)
	s.BorderMetric, s.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.Border)
	s.Resolution = resolveResolution(cfg)
	s.Labels = resolveLabels(cfg)

	s.Common().Requested = collectRequestedMetrics(s.Size, cfg.Fill, cfg.Border)

	return nil
}

func resolveResolution(cfg *config.Spiral) Resolution {
	if r := stages.PtrString(cfg.Resolution); r == "hourly" {
		return Hourly
	}

	return Daily
}

func resolveLabels(cfg *config.Spiral) LabelMode {
	if lbl := stages.PtrString(cfg.Labels); lbl != "" {
		return LabelMode(lbl)
	}

	return LabelLaps
}

// collectRequestedMetrics merges size + fill + border into a deduplicated
// metric set. When size is empty (spiral defaults to commit count), only fill
// and border contribute.
func collectRequestedMetrics(size metric.Name, fill, border *config.MetricSpec) []metric.Name {
	if size != "" {
		return stages.CollectRequestedMetrics(size, fill, border)
	}

	seen := map[metric.Name]bool{}

	var names []metric.Name

	for _, spec := range []*config.MetricSpec{fill, border} {
		if spec != nil && spec.Metric != "" && !seen[spec.Metric] {
			seen[spec.Metric] = true
			names = append(names, spec.Metric)
		}
	}

	return names
}

// BuildTimeBucketsStage builds time buckets from Common().FileTimeRange and
// distributes files into them from Common().FileHistory.
func BuildTimeBucketsStage(s *State) error {
	c := s.Common()

	tr := stages.CommitTimeRange(c.FileTimeRange)
	if tr.Earliest.IsZero() {
		return eris.New("no commit timestamps available to build time buckets")
	}

	buckets := BuildTimeBuckets(s.Resolution, tr.Earliest, tr.Latest)
	if len(buckets) == 0 {
		return eris.New("no time buckets created from commit time range")
	}

	AssignFilesToBuckets(buckets, c.FileHistory)

	s.Buckets = buckets

	return nil
}

// AggregateBucketMetricsStage fills in per-bucket aggregated metric values.
func AggregateBucketMetricsStage(s *State) error {
	AggregateBucketMetrics(s.Buckets, s.Size, s.FillMetric, s.BorderMetric)

	return nil
}

// BuildInksStage builds spiral inks and emits the Rendering image log line.
func BuildInksStage(s *State) error {
	c := s.Common()

	s.Inks = BuildInks(s.Buckets, s.FillMetric, s.FillPalette, s.BorderMetric, s.BorderPalette)

	slog.Info("Rendering image", "output", c.Output, "width", c.Width, "height", c.Height)

	return nil
}

// BuildLegendStage builds the legend config from the inks.
func BuildLegendStage(s *State) error {
	pos, orient := legend.ResolveOptions(
		stages.PtrString(s.Config.Legend),
		stages.PtrString(s.Config.LegendOrientation),
	)

	s.LegendConfig = legend.Build(
		pos, orient,
		s.Inks.Fill, s.FillMetric,
		s.Inks.Border, s.BorderMetric,
		s.Size,
	)

	return nil
}

// LayoutStage runs the spiral layout algorithm and applies disc sizing.
func LayoutStage(s *State) error {
	c := s.Common()

	layout := Layout(s.Buckets, c.Width, c.Height, s.Resolution, s.Labels)
	maxDisc := MaxDiscRadius(len(s.Buckets), c.Width, c.Height, s.Resolution)

	ApplyDiscSizes(layout.Nodes, s.Buckets, maxDisc)

	s.Layout = layout

	return nil
}

// RenderStage renders the spiral to a canvas and attaches the legend.
func RenderStage(s *State) error {
	c := s.Common()

	cv := RenderToCanvas(s.Layout, s.Buckets, c.Width, c.Height, s.Inks)

	if s.LegendConfig != nil {
		cv.SetLegend(*s.LegendConfig)
	}

	c.Canvas = cv

	return nil
}

// LogResult logs the final summary line, matching today's `Rendered spiral …`.
func LogResult(s *State) error {
	c := s.Common()

	files, dirs := stages.CountAll(c.Root)

	slog.Info(
		"Rendered spiral",
		"files", files,
		"directories", dirs,
		"width", c.Width,
		"height", c.Height,
		"size_metric", string(s.Size),
		"fill_metric", string(s.FillMetric),
		"fill_palette", string(s.FillPalette),
		"border_metric", string(s.BorderMetric),
		"border_palette", string(s.BorderPalette),
	)

	return nil
}

var (
	_ pipeline.Stage[*State] = ResolveMetrics
	_ pipeline.Stage[*State] = BuildTimeBucketsStage
	_ pipeline.Stage[*State] = AggregateBucketMetricsStage
	_ pipeline.Stage[*State] = BuildInksStage
	_ pipeline.Stage[*State] = BuildLegendStage
	_ pipeline.Stage[*State] = LayoutStage
	_ pipeline.Stage[*State] = RenderStage
	_ pipeline.Stage[*State] = LogResult
)
