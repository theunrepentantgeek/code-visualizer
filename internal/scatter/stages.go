package scatter

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves scatter axes, size, fill, and border settings.
func ResolveMetrics(s *State) error {
	cfg := s.Config

	xAxis, err := resolveAxisSpec(cfg.XAxis)
	if err != nil {
		return err
	}

	yAxis, err := resolveAxisSpec(cfg.YAxis)
	if err != nil {
		return err
	}

	size := metric.Name(stages.PtrString(cfg.Size))
	if size == "" {
		return eris.New("size metric is required")
	}

	s.XAxis = xAxis
	s.YAxis = yAxis
	s.Size = size
	s.FillMetric = resolveFillMetric(cfg, size)
	s.FillPalette = stages.ResolveFillPalette(cfg.Fill, s.FillMetric)
	s.BorderMetric, s.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.Border)
	s.Common().Requested = collectRequestedMetrics(xAxis.Metric, yAxis.Metric, size, cfg.Fill, cfg.Border)

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
func BuildInksStage(s *State) error {
	c := s.Common()

	s.Dataset = CollectDataset(c.Root, s.XAxis, s.YAxis, s.Size)
	s.Inks = BuildInks(s.Dataset, s.FillMetric, s.FillPalette, s.BorderMetric, s.BorderPalette)

	slog.Info("Rendering image", "output", c.Output, "width", c.Width, "height", c.Height)

	return nil
}

// BuildLegendStage builds the legend config from the resolved inks.
func BuildLegendStage(s *State) error {
	pos, orient := legend.ResolveOptions(
		stages.PtrString(s.Config.Legend),
		stages.PtrString(s.Config.LegendOrientation),
	)

	s.LegendConfig = legend.Build(
		pos,
		orient,
		s.Inks.Fill,
		s.FillMetric,
		s.Inks.Border,
		s.BorderMetric,
		s.Size,
	)

	return nil
}

// LayoutStage positions scatter points within the drawable plot area.
func LayoutStage(s *State) error {
	c := s.Common()
	layoutW, layoutH := legend.ReserveAndLayout(s.LegendConfig, c.Width, c.Height)

	layout := Layout(s.Dataset, layoutW, layoutH, s.XAxis, s.YAxis)
	if layoutW < c.Width || layoutH < c.Height {
		if s.LegendConfig != nil {
			wReduce, hReduce := s.LegendConfig.ReserveSpace()
			dx, dy := legend.LayoutOffset(s.LegendConfig, wReduce, hReduce)
			OffsetLayout(&layout, dx, dy)
		}
	}

	s.Layout = layout

	return nil
}

// RenderStage renders the scatter plot to a canvas.
func RenderStage(s *State) error {
	c := s.Common()

	cv := RenderToCanvas(s.Layout, c.Width, c.Height, s.Inks)
	if s.LegendConfig != nil {
		cv.SetLegend(*s.LegendConfig)
	}

	c.Canvas = cv

	return nil
}

// LogResult logs the final scatter summary.
func LogResult(s *State) error {
	c := s.Common()
	skipped := s.Dataset.Skipped.MissingX + s.Dataset.Skipped.MissingY + s.Dataset.Skipped.MissingSize

	slog.Info(
		"Rendered scatter plot",
		"files", len(s.Dataset.Points),
		"skipped_missing_x", s.Dataset.Skipped.MissingX,
		"skipped_missing_y", s.Dataset.Skipped.MissingY,
		"skipped_missing_size", s.Dataset.Skipped.MissingSize,
		"skipped_total", skipped,
		"output", c.Output,
		"width", c.Width,
		"height", c.Height,
		"x_axis", string(s.XAxis.Metric),
		"y_axis", string(s.YAxis.Metric),
		"size_metric", string(s.Size),
		"fill_metric", string(s.FillMetric),
		"fill_palette", string(s.FillPalette),
		"border_metric", string(s.BorderMetric),
		"border_palette", string(s.BorderPalette),
	)

	return nil
}

var (
	_ pipeline.Stage[*State] = ResolveMetrics
	_ pipeline.Stage[*State] = BuildInksStage
	_ pipeline.Stage[*State] = BuildLegendStage
	_ pipeline.Stage[*State] = LayoutStage
	_ pipeline.Stage[*State] = RenderStage
	_ pipeline.Stage[*State] = LogResult
)
