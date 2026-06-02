package bubbletree

import (
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves size, fill, border metrics + palettes plus the
// label mode, and populates Common().Requested with the metrics the
// scan/provider stages must collect.
func ResolveMetrics(s *State) error {
	cfg := s.Config

	s.Size = metric.Name(stages.PtrString(cfg.Size))
	s.FillMetric = resolveFillMetric(cfg, s.Size)
	s.FillPalette = stages.ResolveFillPalette(cfg.Fill, s.FillMetric)
	s.BorderMetric, s.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.Border)
	s.Labels = resolveLabels(cfg)

	s.Common().Requested = stages.CollectRequestedMetrics(s.Size, cfg.Fill, cfg.Border)

	return nil
}

func resolveFillMetric(cfg *config.Bubbletree, size metric.Name) metric.Name {
	if fill := cfg.Fill.MetricName(); fill != "" {
		return fill
	}

	return size
}

func resolveLabels(cfg *config.Bubbletree) LabelMode {
	if lbl := stages.PtrString(cfg.Labels); lbl != "" {
		return LabelMode(lbl)
	}

	return LabelFoldersOnly
}

// BuildInksStage builds the bubble inks. Also emits the "Rendering image"
// log line preserved from the legacy renderAndLog helper.
func BuildInksStage(s *State) error {
	c := s.Common()

	slog.Info("Rendering image", "output", c.Output, "width", c.Width, "height", c.Height)

	s.Inks = BuildInks(c.Root, s.FillMetric, s.FillPalette, s.BorderMetric, s.BorderPalette)
	if !s.Flat {
		s.Inks.Fill = canvas.NewRadialGradientInk(s.Inks.Fill)
	}

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
		s.Size,
	)

	return nil
}

// LayoutStage reserves legend space, runs the bubble layout algorithm, and
// offsets the result into the remaining canvas area.
func LayoutStage(s *State) error {
	c := s.Common()
	availH := c.Height - stages.EffectiveFooterHeight(c.RootConfig)
	layoutW, layoutH := legend.ReserveAndLayout(s.LegendConfig, c.Width, availH)

	s.Nodes = Layout(c.Root, layoutW, layoutH, s.Size, s.Labels)

	if layoutW < c.Width || layoutH < availH {
		if s.LegendConfig != nil {
			wReduce, hReduce := s.LegendConfig.ReserveSpace()
			dx, dy := legend.LayoutOffset(s.LegendConfig, wReduce, hReduce)
			OffsetNodes(&s.Nodes, dx, dy)
		}
	}

	return nil
}

// RenderStage renders the bubble tree to a canvas and attaches the legend.
func RenderStage(s *State) error {
	c := s.Common()

	cv := RenderToCanvas(&s.Nodes, c.Root, c.Width, c.Height, s.Inks)
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
		"Rendered bubble tree",
		"files", files,
		"directories", dirs,
		"output", c.Output,
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

// Compile-time checks.
var (
	_ pipeline.Stage[*State] = ResolveMetrics
	_ pipeline.Stage[*State] = BuildInksStage
	_ pipeline.Stage[*State] = BuildLegendStage
	_ pipeline.Stage[*State] = LayoutStage
	_ pipeline.Stage[*State] = RenderStage
	_ pipeline.Stage[*State] = LogResult
)
