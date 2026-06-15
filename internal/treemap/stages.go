package treemap

import (
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves size, fill, and border metrics + palettes and fills
// c.Requested.
func ResolveMetrics(c *stages.CommonState, t *State, cfg *config.Treemap) error {
	t.Size = metric.Name(stages.PtrString(cfg.Size))
	t.FillMetric = resolveFillMetric(cfg)
	t.FillPalette = stages.ResolveFillPalette(cfg.Fill, t.FillMetric)
	t.BorderMetric, t.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.Border)

	c.Requested = stages.CollectRequestedMetrics(t.Size, cfg.Fill, cfg.Border)

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
func BuildInksStage(c *stages.CommonState, t *State) error {
	slog.Info("Rendering image", "output", c.Output, "width", c.Width, "height", c.Height)

	t.Inks = BuildInks(c.Root, c.Requested, t.FillMetric, t.FillPalette, t.BorderMetric, t.BorderPalette)
	if !t.Flat {
		t.Inks.Fill = canvas.NewRadialGradientInk(t.Inks.Fill)
	}

	return nil
}

// BuildLegendStage builds the legend config from inks.
func BuildLegendStage(c *stages.CommonState, t *State, cfg *config.Treemap) error {
	pos, orient := legend.ResolveOptions(
		c.RootConfig.LegendPositionStr(),
		c.RootConfig.LegendOrientationStr(),
	)

	t.LegendConfig = legend.Build(
		pos, orient,
		t.Inks.Fill, t.FillMetric,
		t.Inks.Border, t.BorderMetric,
		t.Size,
	)
	if t.LegendConfig != nil {
		t.LegendConfig.LabelSample = labelSampleLines(labelMetricsFor(t, cfg))
	}

	return nil
}

// LayoutStage reserves legend space, lays out rectangles, and applies the
// resulting offset.
func LayoutStage(c *stages.CommonState, t *State) error {
	bounds := c.DrawingBounds
	availH := bounds.Height()
	layoutW, layoutH := legend.ReserveAndLayout(t.LegendConfig, c.Width, availH)

	rect := Layout(c.Root, layoutW, layoutH, t.Size)

	dx, dy := float64(0), float64(bounds.MinY)

	if layoutW < c.Width || layoutH < availH {
		if t.LegendConfig != nil {
			wReduce, hReduce := t.LegendConfig.ReserveSpace()
			ldx, ldy := legend.LayoutOffset(t.LegendConfig, wReduce, hReduce)
			dx += ldx
			dy += ldy
		}
	}

	OffsetRects(&rect, dx, dy)
	t.Root = rect

	return nil
}

// RenderStage renders the treemap to a canvas and attaches the legend.
func RenderStage(c *stages.CommonState, t *State) error {
	cv := RenderToCanvas(t.Root, c.Root, c.Width, c.Height, t.Inks, t.Size)
	if t.LegendConfig != nil {
		cv.SetLegend(*t.LegendConfig)
	}

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
		"fill_metric", string(t.FillMetric),
		"fill_palette", string(t.FillPalette),
		"border_metric", string(t.BorderMetric),
		"border_palette", string(t.BorderPalette),
	)

	return nil
}
