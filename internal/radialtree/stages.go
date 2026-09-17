package radialtree

import (
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves disc-size, fill, and border metrics + palettes and
// fills c.Requested.
func ResolveMetrics(c *stages.CommonState, r *State, cfg *config.Radial) error {
	r.DiscSize = metric.Name(stages.PtrString(cfg.FileDiscSize))
	r.DirectoryDiscSize = resolveDirectoryMetric(
		metric.Name(stages.PtrString(cfg.DirectoryDiscSize)),
		r.DiscSize,
	)
	r.Fill = stages.ResolveColourEncoding(cfg.FileFill, r.DiscSize)
	r.Border = stages.ResolveColourEncoding(cfg.FileBorder, "")
	directoryFillMetric := resolveDirectoryMetric(cfg.DirectoryFill.MetricName(), r.Fill.Metric)
	r.DirectoryFill = stages.ResolveColourEncoding(cfg.DirectoryFill, directoryFillMetric)
	directoryBorderMetric := resolveDirectoryMetric(cfg.DirectoryBorder.MetricName(), r.Border.Metric)
	r.DirectoryBorder = stages.ResolveColourEncoding(cfg.DirectoryBorder, directoryBorderMetric)
	r.Labels = resolveLabels(cfg)
	r.Grain = resolveGrain(cfg)

	c.Requested = stages.CollectRequestedMetricNames(
		r.DiscSize,
		r.Fill.Metric,
		r.Border.Metric,
		r.DirectoryDiscSize,
		r.DirectoryFill.Metric,
		r.DirectoryBorder.Metric,
	)

	return nil
}

func resolveDirectoryMetric(name, fallback metric.Name) metric.Name {
	if name != "" {
		return name
	}

	expr, err := metric.ParseExpression(string(fallback))
	if err != nil || expr.Aggregation != "" {
		return fallback
	}

	desc, ok := provider.GetBase(expr.Base)
	if !ok {
		return ""
	}

	var aggregation metric.AggregationName

	switch desc.Kind {
	case metric.Quantity:
		aggregation = metric.AggSum
	case metric.Measure:
		aggregation = metric.AggMean
	default:
		aggregation = metric.AggMode
	}

	expression := metric.MetricExpression{Filter: expr.Filter, Base: expr.Base, Aggregation: aggregation}
	if _, err := provider.ResolveExpression(expression, metric.LevelDirectory); err != nil {
		return ""
	}

	return expression.ResultName()
}

func resolveLabels(cfg *config.Radial) LabelMode {
	if lbl := stages.PtrString(cfg.Labels); lbl != "" {
		return LabelMode(lbl)
	}

	return LabelFoldersOnly
}

func resolveGrain(cfg *config.Radial) Grain {
	if grain := stages.PtrString(cfg.Grain); grain != "" {
		return Grain(grain)
	}

	return GrainFile
}

// radialCanvasSize returns the diameter of the square radial content area: the
// smaller of the configured width and the drawing height remaining after any
// title/footer reservation.
func radialCanvasSize(c *stages.CommonState) int {
	return min(c.Width, int(c.DrawingBounds.Height()))
}

// BuildInksStage builds the radial inks and emits the Rendering image log line.
func BuildInksStage(c *stages.CommonState, r *State) error {
	canvasSize := radialCanvasSize(c)

	slog.Info("Rendering image", "output", c.Output, "canvas_size", canvasSize)

	r.Inks = BuildInks(c.Root, c.Requested, r.Fill, r.Border)
	r.Inks.DirectoryFill, r.Inks.DirectoryBorder = buildDirectoryInks(
		c.Root,
		c.Requested,
		r.DirectoryFill.Metric,
		r.DirectoryFill.Palette,
		r.DirectoryBorder.Metric,
		r.DirectoryBorder.Palette,
	)

	return nil
}

func buildDirectoryInks(
	root *model.Directory,
	requested stages.RequestedMetrics,
	fillMetric metric.Name,
	fillPalette palette.PaletteName,
	borderMetric metric.Name,
	borderPalette palette.PaletteName,
) (fill inks.Ink, border inks.Ink) {
	fillDesc, _ := requested.DescriptorFor(fillMetric)
	fill = inks.BuildDirectoryMetricInk(root, fillDesc, fillPalette, defaultDirFill)

	borderDesc, _ := requested.DescriptorFor(borderMetric)
	border = inks.BuildDirectoryMetricInk(root, borderDesc, borderPalette, defaultBorder)

	return fill, border
}

// BuildLegendStage builds the legend config from inks.
// Directory grain draws only directory discs, so the legend describes the
// aggregated directory metrics instead of the file ones.
func BuildLegendStage(c *stages.CommonState, r *State) error {
	pos, orient := legend.ResolveOptions(
		c.RootConfig.LegendPositionStr(),
		c.RootConfig.LegendOrientationStr(),
	)

	builder := legend.Builder{
		Position: pos, Orientation: orient,
		FillInk: r.Inks.Fill, FillMetric: r.Fill.Metric,
		BorderInk: r.Inks.Border, BorderMetric: r.Border.Metric,
		SizeMetric: r.DiscSize,
	}

	if r.Grain == GrainDirectory {
		builder.FillInk, builder.FillMetric = r.Inks.DirectoryFill, r.DirectoryFill.Metric
		builder.BorderInk, builder.BorderMetric = r.Inks.DirectoryBorder, r.DirectoryBorder.Metric
		builder.SizeMetric = r.DirectoryDiscSize
	}

	r.LegendConfig = builder.Build()

	return nil
}

// LayoutStage runs the radial tree layout algorithm.
// The circular content is sized to radialCanvasSize (the smaller of the width
// and the drawing height); the surrounding canvas may be non-square.
func LayoutStage(c *stages.CommonState, r *State) error {
	canvasSize := radialCanvasSize(c)
	root := c.Root

	if c.RootConfig != nil &&
		c.RootConfig.Radial != nil &&
		c.RootConfig.Radial.MaxLayers != nil &&
		*c.RootConfig.Radial.MaxLayers > 0 {
		root = model.PruneLayers(c.Root, *c.RootConfig.Radial.MaxLayers)
	}

	r.DisplayRoot = root
	r.Nodes = Layout(root, canvasSize, r.DiscSize, r.DirectoryDiscSize, r.Labels, r.Grain)

	return nil
}

// RenderStage renders the radial tree to a canvas and attaches the legend.
func RenderStage(c *stages.CommonState, r *State) error {
	size := radialCanvasSize(c)
	cx := float64(c.Width) / 2.0
	cy := float64(size)/2.0 + c.DrawingBounds.Min.Y
	root := c.Root

	if r.DisplayRoot != nil {
		root = r.DisplayRoot
	}

	cv := RenderToCanvas(&r.Nodes, root, c.Width, c.Height, cx, cy, r.Inks)
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
		"grain", string(r.Grain),
		"disc_metric", string(r.DiscSize),
		"fill_metric", string(r.Fill.Metric),
		"fill_palette", string(r.Fill.Palette),
		"border_metric", string(r.Border.Metric),
		"border_palette", string(r.Border.Palette),
	)

	return nil
}
