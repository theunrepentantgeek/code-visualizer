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
	directoryDiscSize := &config.MetricSpec{Metric: metric.Name(stages.PtrString(cfg.DirectoryDiscSize))}
	r.DirectoryDiscSize = resolveDirectoryMetric(directoryDiscSize, r.DiscSize)
	r.FillMetric = resolveFillMetric(cfg, r.DiscSize)
	r.FillPalette = stages.ResolveFillPalette(cfg.FileFill, r.FillMetric)
	r.BorderMetric, r.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.FileBorder)
	r.DirectoryFillMetric = resolveDirectoryMetric(cfg.DirectoryFill, r.FillMetric)
	r.DirectoryFillPalette = stages.ResolveFillPalette(
		directoryMetricSpec(cfg.DirectoryFill, r.DirectoryFillMetric),
		r.DirectoryFillMetric,
	)
	r.DirectoryBorderMetric = resolveDirectoryMetric(cfg.DirectoryBorder, r.BorderMetric)
	r.DirectoryBorderPalette = stages.ResolveFillPalette(
		directoryMetricSpec(cfg.DirectoryBorder, r.DirectoryBorderMetric),
		r.DirectoryBorderMetric,
	)
	r.Labels = resolveLabels(cfg)
	r.Grain = resolveGrain(cfg)

	c.Requested = stages.CollectRequestedMetrics(
		r.DiscSize,
		cfg.FileFill,
		cfg.FileBorder,
		directoryMetricSpec(directoryDiscSize, r.DirectoryDiscSize),
		directoryMetricSpec(cfg.DirectoryFill, r.DirectoryFillMetric),
		directoryMetricSpec(cfg.DirectoryBorder, r.DirectoryBorderMetric),
	)

	return nil
}

func resolveFillMetric(cfg *config.Radial, discSize metric.Name) metric.Name {
	if fill := cfg.FileFill.MetricName(); fill != "" {
		return fill
	}

	return discSize
}

func resolveDirectoryMetric(spec *config.MetricSpec, fallback metric.Name) metric.Name {
	if name := spec.MetricName(); name != "" {
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

func directoryMetricSpec(spec *config.MetricSpec, fallback metric.Name) *config.MetricSpec {
	if spec.MetricName() != "" || fallback == "" {
		return spec
	}

	return &config.MetricSpec{Metric: fallback}
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

	r.Inks = BuildInks(c.Root, c.Requested, r.FillMetric, r.FillPalette, r.BorderMetric, r.BorderPalette)
	r.Inks.DirectoryFill, r.Inks.DirectoryBorder = buildDirectoryInks(
		c.Root,
		c.Requested,
		r.DirectoryFillMetric,
		r.DirectoryFillPalette,
		r.DirectoryBorderMetric,
		r.DirectoryBorderPalette,
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
		FillInk: r.Inks.Fill, FillMetric: r.FillMetric,
		BorderInk: r.Inks.Border, BorderMetric: r.BorderMetric,
		SizeMetric: r.DiscSize,
	}

	if r.Grain == GrainDirectory {
		builder.FillInk, builder.FillMetric = r.Inks.DirectoryFill, r.DirectoryFillMetric
		builder.BorderInk, builder.BorderMetric = r.Inks.DirectoryBorder, r.DirectoryBorderMetric
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
		"fill_metric", string(r.FillMetric),
		"fill_palette", string(r.FillPalette),
		"border_metric", string(r.BorderMetric),
		"border_palette", string(r.BorderPalette),
	)

	return nil
}
