package scatter

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves scatter axes, size, fill, and border settings.
func ResolveMetrics(c *stages.CommonState, x *State, cfg *config.Scatter) error {
	if stages.PtrString(cfg.XAxis) == "" {
		return eris.New("x-axis metric is required")
	}

	xAxis, err := resolveAxisSpec(cfg.XAxis)
	if err != nil {
		return eris.Wrap(err, "invalid x-axis metric")
	}

	if stages.PtrString(cfg.YAxis) == "" {
		return eris.New("y-axis metric is required")
	}

	yAxis, err := resolveAxisSpec(cfg.YAxis)
	if err != nil {
		return eris.Wrap(err, "invalid y-axis metric")
	}

	size := metric.Name(stages.PtrString(cfg.Size))
	if size == "" {
		return eris.New("size metric is required")
	}

	x.XAxis = xAxis
	x.YAxis = yAxis
	x.Size = size
	x.FillMetric = resolveFillMetric(cfg, size)
	x.FillPalette = stages.ResolveFillPalette(cfg.Fill, x.FillMetric)
	x.BorderMetric, x.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.Border)
	c.Requested = collectRequestedMetrics(xAxis.Metric, yAxis.Metric, size, cfg.Fill, cfg.Border)

	return nil
}

func resolveAxisSpec(name *string) (AxisSpec, error) {
	metricName := metric.Name(stages.PtrString(name))
	descriptor, ok := provider.GetDescriptor(metricName)

	if !ok {
		return AxisSpec{}, eris.Errorf("unknown axis metric %q", metricName)
	}

	return AxisSpec{Metric: metricName, Kind: descriptor.Kind}, nil
}

func resolveFillMetric(cfg *config.Scatter, size metric.Name) metric.Name {
	if fill := cfg.Fill.MetricName(); fill != "" {
		return fill
	}

	return size
}

func collectRequestedMetrics(xAxis, yAxis, size metric.Name, fill, border *config.MetricSpec) []metric.Name {
	seen := map[metric.Name]bool{}
	names := make([]metric.Name, 0, 5)

	for _, name := range []metric.Name{xAxis, yAxis, size, fill.MetricName(), border.MetricName()} {
		if name == "" || seen[name] {
			continue
		}

		seen[name] = true
		names = append(names, name)
	}

	return names
}

// BuildInksStage collects plottable files and creates point inks.
func BuildInksStage(c *stages.CommonState, x *State) error {
	x.Dataset = CollectDataset(c.Root, x.XAxis, x.YAxis, x.Size)
	x.Inks = BuildInks(x.Dataset, x.FillMetric, x.FillPalette, x.BorderMetric, x.BorderPalette)

	slog.Info("Rendering image", "output", c.Output, "width", c.Width, "height", c.Height)

	return nil
}

// BuildLegendStage builds the legend config from the resolved inks.
func BuildLegendStage(c *stages.CommonState, x *State) error {
	pos, orient := legend.ResolveOptions(
		c.RootConfig.LegendPositionStr(),
		c.RootConfig.LegendOrientationStr(),
	)

	x.LegendConfig = legend.Build(
		pos,
		orient,
		x.Inks.Fill,
		x.FillMetric,
		x.Inks.Border,
		x.BorderMetric,
		x.Size,
	)

	return nil
}

// LayoutStage positions scatter points within the drawable plot area.
func LayoutStage(c *stages.CommonState, x *State) error {
	titleH := stages.EffectiveTitleHeight(c.RootConfig)
	availH := c.Height - stages.EffectiveFooterHeight(c.RootConfig) - titleH
	layoutW, layoutH := legend.ReserveAndLayout(x.LegendConfig, c.Width, availH)

	layout := Layout(x.Dataset, layoutW, layoutH, x.XAxis, x.YAxis)

	dx, dy := float64(0), float64(titleH)

	if layoutW < c.Width || layoutH < availH {
		if x.LegendConfig != nil {
			wReduce, hReduce := x.LegendConfig.ReserveSpace()
			ldx, ldy := legend.LayoutOffset(x.LegendConfig, wReduce, hReduce)
			dx += ldx
			dy += ldy
		}
	}

	OffsetLayout(&layout, dx, dy)
	x.Layout = layout

	return nil
}

// RenderStage renders the scatter plot to a canvas.
func RenderStage(c *stages.CommonState, x *State) error {
	cv := RenderToCanvas(x.Layout, c.Width, c.Height, x.Inks)
	if x.LegendConfig != nil {
		cv.SetLegend(*x.LegendConfig)
	}

	c.Canvas = cv

	return nil
}

// LogResult logs the final scatter summary.
func LogResult(c *stages.CommonState, x *State) error {
	skipped := x.Dataset.Skipped.MissingX + x.Dataset.Skipped.MissingY + x.Dataset.Skipped.MissingSize

	slog.Info(
		"Rendered scatter plot",
		"files", len(x.Dataset.Points),
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
