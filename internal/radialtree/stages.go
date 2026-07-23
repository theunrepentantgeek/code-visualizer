package radialtree

import (
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves disc-size, fill, and border metrics + palettes and
// fills c.Requested.
func ResolveMetrics(c *stages.CommonState, r *State, cfg *config.Radial) error {
	r.DiscSize = metric.Name(stages.PtrString(cfg.DiscSize))
	r.FillMetric = resolveFillMetric(cfg, r.DiscSize)
	r.FillPalette = stages.ResolveFillPalette(cfg.Fill, r.FillMetric)
	r.BorderMetric, r.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.Border)
	r.Labels = resolveLabels(cfg)

	c.Requested = stages.CollectRequestedMetrics(r.DiscSize, cfg.Fill, cfg.Border)

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

	return LabelFoldersOnly
}

// radialCanvasSize returns the diameter of the square radial content area: the
// smaller of the configured width and the drawing height remaining after any
// title/footer reservation.
func radialCanvasSize(c *stages.CommonState) int {
	return min(c.Width, c.DrawingBounds.Height())
}

// BuildInksStage builds the radial inks and emits the Rendering image log line.
func BuildInksStage(c *stages.CommonState, r *State) error {
	canvasSize := radialCanvasSize(c)

	slog.Info("Rendering image", "output", c.Output, "canvas_size", canvasSize)

	r.Inks = BuildInks(c.Root, c.Requested, r.FillMetric, r.FillPalette, r.BorderMetric, r.BorderPalette)

	return nil
}

// BuildLegendStage builds the legend config from inks.
func BuildLegendStage(c *stages.CommonState, r *State) error {
	pos, orient := legend.ResolveOptions(
		c.RootConfig.LegendPositionStr(),
		c.RootConfig.LegendOrientationStr(),
	)
	r.LegendConfig = legend.Build(
		pos, orient,
		r.Inks.Fill, r.FillMetric,
		r.Inks.Border, r.BorderMetric,
		r.DiscSize,
	)

	return nil
}

// LayoutStage runs the radial tree layout algorithm.
// The circular content is sized to radialCanvasSize (the smaller of the width
// and the drawing height); the surrounding canvas may be non-square.
func LayoutStage(c *stages.CommonState, r *State) error {
	canvasSize := radialCanvasSize(c)

	r.Nodes = Layout(c.Root, canvasSize, r.DiscSize, r.Labels)

	return nil
}

// RenderStage renders the radial tree to a canvas and attaches the legend.
func RenderStage(c *stages.CommonState, r *State) error {
	size := radialCanvasSize(c)
	cx := float64(c.Width) / 2.0
	cy := float64(size)/2.0 + float64(c.DrawingBounds.MinY)

	cv := RenderToCanvas(&r.Nodes, c.Root, c.Width, c.Height, cx, cy, r.Inks)
	legend.RenderInto(cv, r.LegendConfig)

	c.Canvas = cv

	return nil
}

// LogResult logs the final summary.
func LogResult(c *stages.CommonState, r *State) error {
	files, dirs := stages.CountAll(c.Root)
	canvasSize := radialCanvasSize(c)

	slog.Info(
		"Rendered radial tree",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"canvas_size", canvasSize,
		"disc_metric", string(r.DiscSize),
		"fill_metric", string(r.FillMetric),
		"fill_palette", string(r.FillPalette),
		"border_metric", string(r.BorderMetric),
		"border_palette", string(r.BorderPalette),
	)

	return nil
}
