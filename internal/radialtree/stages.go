package radialtree

import (
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves disc-size, fill, and border metrics + palettes
// and fills Common().Requested with the metrics the pipeline must collect.
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

// BuildInksStage builds the radial inks and emits the "Rendering image" log line.
func BuildInksStage(s *State) error {
	c := s.Common()
	canvasSize := min(c.Width, c.Height)

	slog.Info("Rendering image", "output", c.Output, "canvas_size", canvasSize)

	s.Inks = BuildInks(c.Root, s.FillMetric, s.FillPalette, s.BorderMetric, s.BorderPalette)

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

// LayoutStage runs the radial tree layout algorithm.
// Radial uses a square canvas: canvasSize = min(Width, Height).
func LayoutStage(s *State) error {
	c := s.Common()
	canvasSize := min(c.Width, c.Height)

	s.Nodes = Layout(c.Root, canvasSize, s.DiscSize, s.Labels)

	return nil
}

// RenderStage renders the radial tree to a canvas and attaches the legend.
func RenderStage(s *State) error {
	c := s.Common()
	canvasSize := min(c.Width, c.Height)

	cv := RenderToCanvas(&s.Nodes, c.Root, canvasSize, s.Inks)
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
	canvasSize := min(c.Width, c.Height)

	slog.Info(
		"Rendered radial tree",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"canvas_size", canvasSize,
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
	_ pipeline.Stage[*State] = BuildInksStage
	_ pipeline.Stage[*State] = BuildLegendStage
	_ pipeline.Stage[*State] = LayoutStage
	_ pipeline.Stage[*State] = RenderStage
	_ pipeline.Stage[*State] = LogResult
)
