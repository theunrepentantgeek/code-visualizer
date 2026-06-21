package bubbletree

import (
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves size/fill/border metrics + palettes plus label mode
// and populates c.Requested.
func ResolveMetrics(c *stages.CommonState, b *State, cfg *config.Bubbletree) error {
	b.Size = metric.Name(stages.PtrString(cfg.Size))
	b.FillMetric = resolveFillMetric(cfg, b.Size)
	b.FillPalette = stages.ResolveFillPalette(cfg.Fill, b.FillMetric)
	b.BorderMetric, b.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.Border)
	b.Labels = resolveLabels(cfg)

	c.Requested = stages.CollectRequestedMetrics(b.Size, cfg.Fill, cfg.Border)

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

// BuildInksStage builds the bubble inks and emits the "Rendering image" log line.
func BuildInksStage(c *stages.CommonState, b *State) error {
	slog.Info("Rendering image", "output", c.Output, "width", c.Width, "height", c.Height)

	b.Inks = BuildInks(c.Root, c.Requested, b.FillMetric, b.FillPalette, b.BorderMetric, b.BorderPalette)
	if !b.Flat {
		b.Inks.Fill = inks.NewRadialGradientInk(b.Inks.Fill)
	}

	return nil
}

// BuildLegendStage builds the legend config from inks.
func BuildLegendStage(c *stages.CommonState, b *State) error {
	pos, orient := legend.ResolveOptions(
		c.RootConfig.LegendPositionStr(),
		c.RootConfig.LegendOrientationStr(),
	)
	b.LegendConfig = legend.Build(
		pos, orient,
		b.Inks.Fill, b.FillMetric,
		b.Inks.Border, b.BorderMetric,
		b.Size,
	)

	return nil
}

// LayoutStage reserves legend space, runs the bubble layout algorithm, and
// offsets the result into the remaining canvas area.
func LayoutStage(c *stages.CommonState, b *State) error {
	bounds := c.DrawingBounds
	availH := bounds.Height()
	layoutW, layoutH := legend.ReserveAndLayout(b.LegendConfig, c.Width, availH)

	b.Nodes = Layout(c.Root, layoutW, layoutH, b.Size, b.Labels)

	dx, dy := float64(0), float64(bounds.MinY)

	if layoutW < c.Width || layoutH < availH {
		if b.LegendConfig != nil {
			wReduce, hReduce := b.LegendConfig.ReserveSpace()
			ldx, ldy := legend.LayoutOffset(b.LegendConfig, wReduce, hReduce)
			dx += ldx
			dy += ldy
		}
	}

	OffsetNodes(&b.Nodes, dx, dy)

	return nil
}

// RenderStage renders the bubble tree to a canvas and attaches the legend.
func RenderStage(c *stages.CommonState, b *State) error {
	cv := RenderToCanvas(&b.Nodes, c.Root, c.Width, c.Height, b.Inks)
	legend.RenderInto(cv, b.LegendConfig)

	c.Canvas = cv

	return nil
}

// LogResult logs the final summary.
func LogResult(c *stages.CommonState, b *State) error {
	files, dirs := stages.CountAll(c.Root)

	slog.Info(
		"Rendered bubble tree",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"width", c.Width,
		"height", c.Height,
		"size_metric", string(b.Size),
		"fill_metric", string(b.FillMetric),
		"fill_palette", string(b.FillPalette),
		"border_metric", string(b.BorderMetric),
		"border_palette", string(b.BorderPalette),
	)

	return nil
}
