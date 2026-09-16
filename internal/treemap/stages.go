package treemap

import (
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves size, fill, and border metrics + palettes and fills
// c.Requested.
func ResolveMetrics(c *stages.CommonState, t *State, cfg *config.Treemap) error {
	t.Size = metric.Name(stages.PtrString(cfg.Size))
	t.Fill = stages.ResolveColourEncoding(cfg.Fill, t.Size)
	t.Border = stages.ResolveColourEncoding(cfg.Border, "")

	c.Requested = stages.CollectRequestedMetrics(t.Size, cfg.Fill, cfg.Border)

	return nil
}

// BuildInksStage builds the treemap inks. Also emits the "Rendering image"
// log line preserved from the legacy renderAndLog helper.
func BuildInksStage(c *stages.CommonState, t *State) error {
	slog.Info("Rendering image", "output", c.Output, "width", c.Width, "height", c.Height)

	t.Inks = BuildInks(c.Root, c.Requested, t.Fill, t.Border)
	if !t.Flat {
		t.Inks.Fill = inks.NewRadialGradientInk(t.Inks.Fill)
	}

	return nil
}

// BuildLegendStage builds the legend config from inks.
func BuildLegendStage(c *stages.CommonState, t *State, cfg *config.Treemap) error {
	pos, orient := legend.ResolveOptions(
		c.RootConfig.LegendPositionStr(),
		c.RootConfig.LegendOrientationStr(),
	)

	t.LegendConfig = legend.Builder{
		Position: pos, Orientation: orient,
		FillInk: t.Inks.Fill, FillMetric: t.Fill.Metric,
		BorderInk: t.Inks.Border, BorderMetric: t.Border.Metric,
		SizeMetric: t.Size,
	}.Build()
	if t.LegendConfig != nil {
		t.LegendConfig.LabelSample = legend.LabelSample{
			Shape: legend.LabelSampleSquare,
			Lines: labelSampleLines(labelMetricsFor(t, cfg)),
		}
	}

	return nil
}

// LayoutStage reserves legend space, lays out rectangles, and applies the
// resulting offset.
func LayoutStage(c *stages.CommonState, t *State) error {
	bounds := c.DrawingBounds
	availH := int(bounds.Height())
	layoutW, layoutH := legend.ReserveAndLayout(t.LegendConfig, c.Width, availH)

	rect := Layout(c.Root, layoutW, layoutH, t.Size)

	dx, dy := float64(0), bounds.Min.Y

	if layoutW < c.Width || layoutH < availH {
		if t.LegendConfig != nil {
			reserved := t.LegendConfig.ReserveSpace()
			ldx, ldy := legend.LayoutOffset(t.LegendConfig, reserved)
			dx += ldx
			dy += ldy
		}
	}

	OffsetRects(&rect, geometry.NewVector(dx, dy))
	t.Root = rect

	return nil
}

// RenderStage renders the treemap to a canvas and attaches the legend.
func RenderStage(c *stages.CommonState, t *State) error {
	cv := RenderToCanvas(t.Root, c.Root, c.Width, c.Height, t.Inks, t.Size)
	legend.RenderInto(cv, t.LegendConfig)

	slog.Debug("rendering", "width", c.Width, "height", c.Height, "output", c.Output)

	c.Canvas = cv

	return nil
}

// LabelStage builds the reusable block labels for treemap file rectangles.
func LabelStage(c *stages.CommonState, t *State, cfg *config.Treemap) error {
	t.BlockLabels = buildBlockLabels(t.Root, c.Root, t.Inks.Fill, labelMetricsFor(t, cfg))

	return nil
}

func labelMetricsFor(t *State, cfg *config.Treemap) LabelMetrics {
	return LabelMetrics{
		Size:   t.Size,
		Fill:   cfg.Fill.MetricName(),
		Border: cfg.Border.MetricName(),
	}
}

// LogResult logs the final summary.
func LogResult(c *stages.CommonState, t *State) error {
	files, dirs := stages.CountAll(c.Root)

	slog.Info(
		"Rendered treemap",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"width", c.Width,
		"height", c.Height,
		"size_metric", string(t.Size),
		"fill_metric", string(t.Fill.Metric),
		"fill_palette", string(t.Fill.Palette),
		"border_metric", string(t.Border.Metric),
		"border_palette", string(t.Border.Palette),
	)

	return nil
}
