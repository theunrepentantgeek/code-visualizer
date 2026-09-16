package donuttree

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

// ResolveMetrics resolves directory aggregation expressions and palettes.
func ResolveMetrics(c *stages.CommonState, d *State, cfg *config.DonutTree) error {
	sizeMetric, err := resolveDirectoryMetric(metric.Name(stages.PtrString(cfg.Size)))
	if err != nil {
		return eris.Wrap(err, "invalid size metric")
	}

	d.SizeMetric = sizeMetric

	fillBase := cfg.Fill.MetricName()
	if fillBase == "" {
		fillBase = metric.Name(stages.PtrString(cfg.Size))
	}

	fillMetric, err := resolveDirectoryMetric(fillBase)
	if err != nil {
		return eris.Wrap(err, "invalid fill metric")
	}

	d.Fill = viz.ColourEncoding{Metric: fillMetric, Palette: stages.ResolveFillPalette(cfg.Fill, fillMetric)}

	if borderBase := cfg.Border.MetricName(); borderBase != "" {
		borderMetric, resolveErr := resolveDirectoryMetric(borderBase)
		err = resolveErr
		if err != nil {
			return eris.Wrap(err, "invalid border metric")
		}

		d.Border = viz.ColourEncoding{
			Metric: borderMetric, Palette: stages.ResolveFillPalette(cfg.Border, borderMetric),
		}
	}

	c.Requested = stages.CollectRequestedMetrics(
		d.SizeMetric,
		effectiveMetricSpec(cfg.Fill, d.Fill.Metric),
		effectiveMetricSpec(cfg.Border, d.Border.Metric),
	)

	return nil
}

func resolveDirectoryMetric(name metric.Name) (metric.Name, error) {
	expr, err := metric.ParseExpression(string(name))
	if err != nil {
		return "", eris.Wrap(err, "parse metric expression")
	}

	desc, ok := provider.GetBase(expr.Base)
	if !ok {
		return "", eris.Errorf("unknown base metric %q", expr.Base)
	}

	if expr.Aggregation.IsZero() {
		expr.Aggregation = aggregationForKind(desc.Kind)
	}

	if _, err := provider.ResolveExpression(expr, metric.LevelDirectory); err != nil {
		return "", eris.Wrap(err, "resolve metric expression")
	}

	return expr.ResultName(), nil
}

func aggregationForKind(kind metric.Kind) metric.AggregationName {
	switch kind {
	case metric.Quantity:
		return metric.AggSum
	case metric.Measure:
		return metric.AggMean
	default:
		return metric.AggMode
	}
}

func effectiveMetricSpec(spec *config.MetricSpec, name metric.Name) *config.MetricSpec {
	if name == "" {
		return nil
	}

	return &config.MetricSpec{Metric: name, Palette: spec.PaletteName()}
}

// BuildInksStage builds the donut tree's directory inks.
func BuildInksStage(c *stages.CommonState, d *State) error {
	slog.Info("Rendering image", "output", c.Output, "canvas_size", donutCanvasSize(c))

	d.Inks = BuildInks(
		c.Root,
		c.Requested,
		d.Fill.Metric,
		d.Fill.Palette,
		d.Border.Metric,
		d.Border.Palette,
	)

	return nil
}

// BuildLegendStage builds a legend for the effective directory metrics.
func BuildLegendStage(c *stages.CommonState, d *State) error {
	pos, orient := legend.ResolveOptions(
		c.RootConfig.LegendPositionStr(),
		c.RootConfig.LegendOrientationStr(),
	)

	d.LegendConfig = legend.Builder{
		Position:     pos,
		Orientation:  orient,
		FillInk:      d.Inks.Fill,
		FillMetric:   d.Fill.Metric,
		BorderInk:    d.Inks.Border,
		BorderMetric: d.Border.Metric,
		SizeMetric:   d.SizeMetric,
	}.Build()
	if d.LegendConfig != nil {
		var cfg *config.DonutTree
		if c.RootConfig != nil {
			cfg = c.RootConfig.DonutTree
		}

		d.LegendConfig.LabelSample = legend.LabelSample{
			Shape: legend.LabelSampleArc,
			Lines: labelSampleLines(labelMetricsFor(d, cfg)),
		}
	}

	return nil
}

// LayoutStage lays out directory sectors within the square drawing area.
func LayoutStage(c *stages.CommonState, d *State) error {
	root := c.Root

	if c.RootConfig != nil &&
		c.RootConfig.DonutTree != nil &&
		c.RootConfig.DonutTree.MaxLayers != nil &&
		*c.RootConfig.DonutTree.MaxLayers > 0 {
		root = model.PruneLayers(c.Root, *c.RootConfig.DonutTree.MaxLayers)
	}

	d.DisplayRoot = root
	d.Layout = Layout(root, donutCanvasSize(c), d.SizeMetric)

	return nil
}

func donutCanvasSize(c *stages.CommonState) int {
	return min(c.Width, int(c.DrawingBounds.Height()))
}

// RenderStage renders the donut tree into its reserved drawing bounds.
func RenderStage(c *stages.CommonState, d *State) error {
	size := donutCanvasSize(c)
	d.Layout.Center = geometry.NewPoint(
		float64(c.Width)/2,
		c.DrawingBounds.Min.Y+float64(size)/2,
	)

	var cfg *config.DonutTree
	if c.RootConfig != nil {
		cfg = c.RootConfig.DonutTree
	}

	root := c.Root

	if d.DisplayRoot != nil {
		root = d.DisplayRoot
	}

	cv := RenderToCanvas(d.Layout, root, c.Width, c.Height, d.Inks, labelMetricsFor(d, cfg))
	if c.DrawingBounds.Max.Y > 0 {
		cv.SetDrawingBounds(int(c.DrawingBounds.Min.Y), int(c.DrawingBounds.Max.Y))
	}

	legend.RenderInto(cv, d.LegendConfig)
	c.Canvas = cv

	return nil
}

func labelMetricsFor(d *State, cfg *config.DonutTree) LabelMetrics {
	metrics := LabelMetrics{Size: d.SizeMetric}

	if cfg == nil {
		return metrics
	}

	if cfg.Fill != nil && cfg.Fill.MetricName() != "" {
		metrics.Fill = d.Fill.Metric
		metrics.IncludeFill = true
	}

	if cfg.Border != nil && cfg.Border.MetricName() != "" {
		metrics.Border = d.Border.Metric
		metrics.IncludeBorder = true
	}

	return metrics
}

// LogResult logs the final donut tree summary.
func LogResult(c *stages.CommonState, d *State) error {
	files, dirs := stages.CountAll(c.Root)

	slog.Info(
		"Rendered donut tree",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"canvas_size", donutCanvasSize(c),
		"size_metric", string(d.SizeMetric),
		"fill_metric", string(d.Fill.Metric),
		"fill_palette", string(d.Fill.Palette),
		"border_metric", string(d.Border.Metric),
		"border_palette", string(d.Border.Palette),
	)

	return nil
}
