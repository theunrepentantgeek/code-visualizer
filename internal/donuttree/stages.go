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

	d.FillMetric, err = resolveDirectoryMetric(fillBase)
	if err != nil {
		return eris.Wrap(err, "invalid fill metric")
	}

	d.FillPalette = stages.ResolveFillPalette(cfg.Fill, d.FillMetric)

	if borderBase := cfg.Border.MetricName(); borderBase != "" {
		d.BorderMetric, err = resolveDirectoryMetric(borderBase)
		if err != nil {
			return eris.Wrap(err, "invalid border metric")
		}

		d.BorderPalette = stages.ResolveFillPalette(cfg.Border, d.BorderMetric)
	}

	c.Requested = stages.CollectRequestedMetrics(
		d.SizeMetric,
		effectiveMetricSpec(cfg.Fill, d.FillMetric),
		effectiveMetricSpec(cfg.Border, d.BorderMetric),
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
		d.FillMetric,
		d.FillPalette,
		d.BorderMetric,
		d.BorderPalette,
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
		FillMetric:   d.FillMetric,
		BorderInk:    d.Inks.Border,
		BorderMetric: d.BorderMetric,
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
	return min(c.Width, c.DrawingBounds.Height())
}

// RenderStage renders the donut tree into its reserved drawing bounds.
func RenderStage(c *stages.CommonState, d *State) error {
	size := donutCanvasSize(c)
	d.Layout.Center = geometry.Point{
		X: float64(c.Width) / 2,
		Y: float64(c.DrawingBounds.MinY) + float64(size)/2,
	}

	var cfg *config.DonutTree
	if c.RootConfig != nil {
		cfg = c.RootConfig.DonutTree
	}

	root := c.Root

	if d.DisplayRoot != nil {
		root = d.DisplayRoot
	}

	cv := RenderToCanvas(d.Layout, root, c.Width, c.Height, d.Inks, labelMetricsFor(d, cfg))
	if c.DrawingBounds.MaxY > 0 {
		cv.SetDrawingBounds(c.DrawingBounds.MinY, c.DrawingBounds.MaxY)
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
		metrics.Fill = d.FillMetric
		metrics.IncludeFill = true
	}

	if cfg.Border != nil && cfg.Border.MetricName() != "" {
		metrics.Border = d.BorderMetric
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
		"fill_metric", string(d.FillMetric),
		"fill_palette", string(d.FillPalette),
		"border_metric", string(d.BorderMetric),
		"border_palette", string(d.BorderPalette),
	)

	return nil
}
