package treemap

import (
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves size, fill, and border metrics + palettes and
// fills Common().Requested.
func ResolveMetrics(s *State) error {
	cfg := s.Config

	s.Size = metric.Name(stages.PtrString(cfg.Size))
	s.FillMetric = resolveFillMetric(cfg)
	s.FillPalette = stages.ResolveFillPalette(cfg.Fill, s.FillMetric)
	s.BorderMetric, s.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.Border)

	s.Common().Requested = stages.CollectRequestedMetrics(s.Size, cfg.Fill, cfg.Border)

	return nil
}

func resolveFillMetric(cfg *config.Treemap) metric.Name {
	if fill := cfg.Fill.MetricName(); fill != "" {
		return fill
	}

	return metric.Name(stages.PtrString(cfg.Size))
}

// BuildInksStage builds the treemap inks. Also emits the "Rendering image"
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
	if s.LegendConfig != nil {
		s.LegendConfig.LabelSample = labelSampleLines(labelMetricsForState(s))
	}

	return nil
}

// LayoutStage reserves legend space, lays out rectangles, and applies the
// resulting offset.
func LayoutStage(s *State) error {
	c := s.Common()
	availH := c.Height - stages.EffectiveFooterHeight(c.RootConfig)
	layoutW, layoutH := legend.ReserveAndLayout(s.LegendConfig, c.Width, availH)

	rect := Layout(c.Root, layoutW, layoutH, s.Size)

	if layoutW < c.Width || layoutH < availH {
		if s.LegendConfig != nil {
			wReduce, hReduce := s.LegendConfig.ReserveSpace()
			dx, dy := legend.LayoutOffset(s.LegendConfig, wReduce, hReduce)
			OffsetRects(&rect, dx, dy)
		}
	}

	s.Root = rect

	return nil
}

// RenderStage renders the treemap to a canvas and attaches the legend.
func RenderStage(s *State) error {
	c := s.Common()

	cv := RenderToCanvas(s.Root, c.Root, c.Width, c.Height, s.Inks, s.Size)
	if s.LegendConfig != nil {
		cv.SetLegend(*s.LegendConfig)
	}

	slog.Debug("rendering", "width", c.Width, "height", c.Height, "output", c.Output)

	c.Canvas = cv

	return nil
}

// LabelStage builds the reusable block labels for treemap file rectangles.
func LabelStage(s *State) error {
	s.BlockLabels = buildBlockLabels(s.Root, s.Common().Root, s.Inks.Fill, labelMetricsForState(s))

	return nil
}

func labelMetricsForState(s *State) LabelMetrics {
	return LabelMetrics{
		Size:   s.Size,
		Fill:   s.Config.Fill.MetricName(),
		Border: s.Config.Border.MetricName(),
	}
}

// LogResult logs the final summary.
func LogResult(s *State) error {
	c := s.Common()
	files, dirs := stages.CountAll(c.Root)

	slog.Info(
		"Rendered treemap",
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
	_ pipeline.Stage[*State] = LabelStage
	_ pipeline.Stage[*State] = LogResult
)
