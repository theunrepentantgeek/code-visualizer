package scatter

import (
	"errors"
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

// ResolveMetrics resolves scatter axes, size, fill, and border settings.
func ResolveMetrics(c *stages.CommonState, x *State, cfg *config.Scatter) error {
	grain, err := resolveGrain(cfg)
	if err != nil {
		return err
	}

	x.Grain = grain
	level := metricLevelForGrain(grain)

	xAxis, err := resolveRequiredAxis("x-axis", cfg.XAxis, cfg.XScale, level)
	if err != nil {
		return err
	}

	yAxis, err := resolveRequiredAxis("y-axis", cfg.YAxis, cfg.YScale, level)
	if err != nil {
		return err
	}

	size, err := resolveSizeMetric(cfg.Size, level)
	if err != nil {
		return err
	}

	if err := validateColourMetrics(cfg, level); err != nil {
		return err
	}

	x.XAxis = xAxis
	x.YAxis = yAxis
	x.Size = size
	x.FillMetric = resolveFillMetric(cfg, size)
	x.FillPalette = stages.ResolveFillPalette(cfg.Fill, x.FillMetric)
	x.BorderMetric, x.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.Border)
	c.Requested = collectRequestedMetrics(xAxis.Metric, yAxis.Metric, size, cfg.Fill, cfg.Border, level)

	return nil
}

func resolveRequiredAxis(
	label string,
	name, scale *string,
	level metric.MetricLevel,
) (AxisSpec, error) {
	if stages.PtrString(name) == "" {
		return AxisSpec{}, eris.Errorf("%s metric is required", label)
	}

	axis, err := resolveAxisSpec(name, scale, level)
	if err != nil {
		return AxisSpec{}, eris.Wrapf(err, "invalid %s configuration", label)
	}

	return axis, nil
}

func resolveSizeMetric(name *string, level metric.MetricLevel) (metric.Name, error) {
	size := metric.Name(stages.PtrString(name))
	if size == "" {
		return "", eris.New("size metric is required")
	}

	resolved, err := provider.ResolveName(size, level)
	if err != nil {
		return "", eris.Wrapf(err, "invalid size metric %q", size)
	}

	if resolved.ResultKind != metric.Quantity && resolved.ResultKind != metric.Measure {
		return "", eris.Errorf("size metric must be numeric, got %q", size)
	}

	return size, nil
}

func validateColourMetrics(cfg *config.Scatter, level metric.MetricLevel) error {
	specs := []struct {
		label string
		name  metric.Name
	}{
		{label: "fill", name: cfg.Fill.MetricName()},
		{label: "border", name: cfg.Border.MetricName()},
	}

	for _, spec := range specs {
		if spec.name == "" {
			continue
		}

		if _, err := provider.ResolveName(spec.name, level); err != nil {
			return eris.Wrapf(err, "invalid %s metric %q", spec.label, spec.name)
		}
	}

	return nil
}

func resolveAxisSpec(name *string, scale *string, level metric.MetricLevel) (AxisSpec, error) {
	metricName := metric.Name(stages.PtrString(name))

	resolved, err := provider.ResolveName(metricName, level)
	if err != nil {
		return AxisSpec{}, eris.Wrapf(err, "invalid axis metric %q", metricName)
	}

	spec := AxisSpec{Metric: metricName, Kind: resolved.ResultKind}

	scaleStr := stages.PtrString(scale)
	switch scaleStr {
	case "", "linear":
		spec.Scale = Linear
	case "log":
		if resolved.ResultKind == metric.Classification {
			return AxisSpec{}, eris.Errorf(
				"log scale is only valid for numeric metrics; %q is a classification metric",
				metricName,
			)
		}

		spec.Scale = Log
	default:
		return AxisSpec{}, eris.Errorf("unknown scale %q; must be \"linear\" or \"log\"", scaleStr)
	}

	return spec, nil
}

func resolveFillMetric(cfg *config.Scatter, size metric.Name) metric.Name {
	if fill := cfg.Fill.MetricName(); fill != "" {
		return fill
	}

	return size
}

func collectRequestedMetrics(
	xAxis, yAxis, size metric.Name,
	fill, border *config.MetricSpec,
	level metric.MetricLevel,
) stages.RequestedMetrics {
	seen := map[metric.Name]bool{}
	names := make([]metric.Name, 0, 5)

	for _, name := range []metric.Name{xAxis, yAxis, size, fill.MetricName(), border.MetricName()} {
		if name == "" || seen[name] {
			continue
		}

		seen[name] = true
		names = append(names, name)
	}

	return stages.ClassifyRequestedMetrics(names, level)
}

func resolveGrain(cfg *config.Scatter) (viz.Grain, error) {
	switch grain := stages.PtrString(cfg.Grain); grain {
	case "", string(viz.GrainFile):
		return viz.GrainFile, nil
	case string(viz.GrainDirectory):
		return viz.GrainDirectory, nil
	default:
		return "", eris.Errorf("unknown grain %q; must be \"file\" or \"directory\"", grain)
	}
}

func metricLevelForGrain(grain viz.Grain) metric.MetricLevel {
	if grain == viz.GrainDirectory {
		return metric.LevelDirectory
	}

	return metric.LevelFile
}

// BuildInksStage collects plottable files and creates point inks.
func BuildInksStage(c *stages.CommonState, x *State) error {
	x.Dataset = CollectDataset(c.Root, x.Grain, x.XAxis, x.YAxis, x.Size)

	if err := ValidateLogScale(x.Dataset, x.XAxis, x.YAxis); err != nil {
		return err
	}

	x.Inks = BuildInks(x.Dataset, c.Requested, x.FillMetric, x.FillPalette, x.BorderMetric, x.BorderPalette)

	slog.Info("Rendering image", "output", c.Output, "width", c.Width, "height", c.Height)

	return nil
}

// ValidateLogScale checks that all data values are positive when log scale is used.
// Both axes are always validated so that users see all issues in a single run.
func ValidateLogScale(dataset Dataset, xAxis, yAxis AxisSpec) error {
	xValue := func(p PointDatum) float64 { return p.X.Numeric }
	xErr := validateAxisPositive(dataset.Points, xAxis, "x-axis", xValue)

	yValue := func(p PointDatum) float64 { return p.Y.Numeric }
	yErr := validateAxisPositive(dataset.Points, yAxis, "y-axis", yValue)

	return errors.Join(xErr, yErr)
}

func validateAxisPositive(points []PointDatum, axis AxisSpec, label string, value func(PointDatum) float64) error {
	if axis.Scale != Log {
		return nil
	}

	for _, point := range points {
		if value(point) <= 0 {
			return eris.Errorf(
				"log scale on %s requires all values to be positive; node %q has value %g",
				label, point.Name(), value(point),
			)
		}
	}

	return nil
}

// BuildLegendStage builds the legend config from the resolved inks.
func BuildLegendStage(c *stages.CommonState, x *State) error {
	pos, orient := legend.ResolveOptions(
		c.RootConfig.LegendPositionStr(),
		c.RootConfig.LegendOrientationStr(),
	)

	x.LegendConfig = legend.Builder{
		Position: pos, Orientation: orient,
		FillInk: x.Inks.Fill, FillMetric: x.FillMetric,
		BorderInk: x.Inks.Border, BorderMetric: x.BorderMetric,
		SizeMetric: x.Size,
	}.Build()

	return nil
}

// LayoutStage positions scatter points within the drawable plot area.
func LayoutStage(c *stages.CommonState, x *State) error {
	bounds := c.DrawingBounds
	availH := bounds.Height()
	layoutW, layoutH := legend.ReserveAndLayout(x.LegendConfig, c.Width, availH)

	layout := Layout(x.Dataset, layoutW, layoutH, x.XAxis, x.YAxis)

	dx, dy := float64(0), float64(bounds.MinY)

	if layoutW < c.Width || layoutH < availH {
		if x.LegendConfig != nil {
			wReduce, hReduce := x.LegendConfig.ReserveSpace()
			ldx, ldy := legend.LayoutOffset(x.LegendConfig, wReduce, hReduce)
			dx += ldx
			dy += ldy
		}
	}

	OffsetLayout(&layout, geometry.NewVector(dx, dy))
	x.Layout = layout

	return nil
}

// RenderStage renders the scatter plot to a canvas.
func RenderStage(c *stages.CommonState, x *State) error {
	cv := RenderToCanvas(x.Layout, c.Width, c.Height, x.Inks)
	legend.RenderInto(cv, x.LegendConfig)

	c.Canvas = cv

	return nil
}

// LogResult logs the final scatter summary.
func LogResult(c *stages.CommonState, x *State) error {
	skipped := x.Dataset.Skipped.Total()

	slog.Info(
		"Rendered scatter plot",
		"nodes", len(x.Dataset.Points),
		"grain", string(x.Grain),
		"skipped_missing_x", x.Dataset.Skipped.MissingX,
		"skipped_missing_y", x.Dataset.Skipped.MissingY,
		"skipped_missing_size", x.Dataset.Skipped.MissingSize,
		"skipped_total", skipped,
		"output", c.Output,
		"width", c.Width,
		"height", c.Height,
		"x_axis", string(x.XAxis.Metric),
		"y_axis", string(x.YAxis.Metric),
		"size_metric", string(x.Size),
		"fill_metric", string(x.FillMetric),
		"fill_palette", string(x.FillPalette),
		"border_metric", string(x.BorderMetric),
		"border_palette", string(x.BorderPalette),
	)

	return nil
}
