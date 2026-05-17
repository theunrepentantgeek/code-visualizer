package radialtree

import (
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves disc-size, fill, border metrics + palettes plus the
// label mode, and populates Common().Requested with the metrics the
// scan/provider stages must collect.
func ResolveMetrics(s *State) error {
	cfg := s.Config

	s.DiscSize = metric.Name(stages.PtrString(cfg.DiscSize))
	s.FillMetric = resolveFillMetric(cfg, s.DiscSize)
	s.FillPalette = stages.ResolveFillPalette(cfg.Fill, s.FillMetric)
	s.BorderMetric, s.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.Border)
	s.Labels = resolveLabels(cfg)

	s.Common().Requested = stages.CollectRequestedMetrics(s.DiscSize, cfg.Fill, cfg.Border)

	return nil
}

func resolveFillMetric(cfg *config.Radial, discSize metric.Name) metric.Name {
	if fill := cfg.Fill.MetricName(); fill != "" {
		return fill
	}

	return discSize
}

func resolveLabels(cfg *config.Radial) LabelMode {
	if lbl := stages.PtrString(cfg.Labels); lbl != "" {
		return LabelMode(lbl)
	}

	return LabelAll
}

// ResolveCanvasSize computes the square canvas size from the resolved
// width and height. Radial visualizations use a square canvas.
func ResolveCanvasSize(s *State) error {
	c := s.Common()
	s.CanvasSize = min(c.Width, c.Height)

	return nil
}

// BuildInksStage builds the radial inks. Also emits the "Rendering image"
// log line preserved from the legacy renderAndLog helper.
func BuildInksStage(s *State) error {
	slog.Info("Rendering image", "output", s.Common().Output, "canvas_size", s.CanvasSize)

	s.Inks = BuildInks(s.Common().Root, s.FillMetric, s.FillPalette, s.BorderMetric, s.BorderPalette)

	return nil
}

// BuildLegendStage builds the legend config from inks.
func BuildLegendStage(s *State) error {
	pos, orient := legend.ResolveOptions(
		stages.PtrString(s.Config.Legend),
		stages.PtrString(s.Config.LegendOrientation),
	)
	s.LegendConfig = legend.Build(
		pos, orient,
		s.Inks.Fill, s.FillMetric,
		s.Inks.Border, s.BorderMetric,
		s.DiscSize,
	)

	return nil
}

// LayoutStage runs the radial layout algorithm.
func LayoutStage(s *State) error {
	s.Nodes = Layout(s.Common().Root, s.CanvasSize, s.DiscSize, s.Labels)

	return nil
}

// RenderStage renders the radial tree to a canvas and attaches the legend.
func RenderStage(s *State) error {
	c := s.Common()

	cv := RenderToCanvas(&s.Nodes, c.Root, s.CanvasSize, s.Inks)
	if s.LegendConfig != nil {
		cv.SetLegend(*s.LegendConfig)
	}

	c.Canvas = cv

	return nil
}

// LogResult logs the final summary.
func LogResult(s *State) error {
	c := s.Common()
	files, dirs := stages.CountAll(c.Root)

	slog.Info(
		"Rendered radial tree",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"canvas_size", s.CanvasSize,
		"disc_metric", string(s.DiscSize),
		"fill_metric", string(s.FillMetric),
		"fill_palette", string(s.FillPalette),
		"border_metric", string(s.BorderMetric),
		"border_palette", string(s.BorderPalette),
	)

	return nil
}

// Compile-time checks.
var (
	_ pipeline.Stage[*State] = ResolveMetrics
	_ pipeline.Stage[*State] = ResolveCanvasSize
	_ pipeline.Stage[*State] = BuildInksStage
	_ pipeline.Stage[*State] = BuildLegendStage
	_ pipeline.Stage[*State] = LayoutStage
	_ pipeline.Stage[*State] = RenderStage
	_ pipeline.Stage[*State] = LogResult
)
