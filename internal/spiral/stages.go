package spiral

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/surface"
)

// ResolveMetrics resolves metric and resolution settings from the spiral config
// and populates c.Requested.
func ResolveMetrics(c *stages.CommonState, p *State, cfg *config.Spiral) error {
	p.Size = metric.Name(stages.PtrString(cfg.Size))
	p.FillMetric = cfg.Fill.MetricName()
	p.FillPalette = stages.ResolveFillPalette(cfg.Fill, p.FillMetric)
	p.BorderMetric, p.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.Border)
	p.SurfaceEnabled = cfg.SurfaceEnabled()
	p.SurfaceMetric = ""

	p.SurfacePalette = ""
	if p.SurfaceEnabled {
		if cfg.SurfaceMetric != nil && !cfg.SurfaceMetric.IsZero() {
			p.SurfaceMetric = cfg.SurfaceMetric.MetricName()
			p.SurfacePalette = stages.ResolveFillPalette(cfg.SurfaceMetric, p.SurfaceMetric)
		} else {
			p.SurfaceMetric = p.FillMetric
			p.SurfacePalette = p.FillPalette
		}
	}

	p.Resolution = resolveResolution(cfg)

	c.Requested = collectRequestedMetrics(p.Size, cfg.Fill, cfg.Border, cfg.SurfaceMetric)

	return nil
}

func resolveResolution(cfg *config.Spiral) Resolution {
	if r := stages.PtrString(cfg.Resolution); r == "hourly" {
		return Hourly
	}

	return Daily
}

// collectRequestedMetrics merges size, fill, border, and surface into a
// deduplicated metric set. When size is empty (spiral defaults to commit
// count), only configured colour and surface metrics contribute.
func collectRequestedMetrics(
	size metric.Name,
	fill, border, surfaceSpec *config.MetricSpec,
) stages.RequestedMetrics {
	seen := map[metric.Name]bool{}
	names := make([]metric.Name, 0, 4)

	if size != "" {
		seen[size] = true
		names = append(names, size)
	}

	for _, spec := range []*config.MetricSpec{fill, border, surfaceSpec} {
		if spec != nil && spec.Metric != "" && !seen[spec.Metric] {
			seen[spec.Metric] = true
			names = append(names, spec.Metric)
		}
	}

	return stages.ClassifyRequestedMetrics(names, metric.LevelDirectory)
}

// BuildTimeBucketsStage builds time buckets from c.FileTimeRange and
// distributes files into them from c.FileHistory.
func BuildTimeBucketsStage(c *stages.CommonState, p *State) error {
	tr := stages.CommitTimeRange(c.FileTimeRange)
	if tr.Earliest.IsZero() {
		return eris.New("no commit timestamps available to build time buckets")
	}

	buckets := BuildTimeBuckets(p.Resolution, tr.Earliest, tr.Latest)
	if len(buckets) == 0 {
		return eris.New("no time buckets created from commit time range")
	}

	AssignFilesToBuckets(buckets, c.FileHistory)

	p.Buckets = buckets

	return nil
}

// AggregateBucketMetricsStage fills in per-bucket aggregated metric values.
func AggregateBucketMetricsStage(c *stages.CommonState, p *State) error {
	AggregateBucketMetrics(p.Buckets, c.Requested, p.Size, p.FillMetric, p.BorderMetric, p.SurfaceMetric)

	return nil
}

// BuildInksStage builds spiral inks and emits the Rendering image log line.
func BuildInksStage(c *stages.CommonState, p *State) error {
	p.Inks = BuildInks(p.Buckets, c.Requested, p.FillMetric, p.FillPalette, p.BorderMetric, p.BorderPalette)

	p.SurfaceInk = nil
	if p.SurfaceEnabled {
		if p.SurfaceMetric == p.FillMetric && p.SurfacePalette == p.FillPalette {
			p.SurfaceInk = p.Inks.Fill
		} else {
			values := make([]float64, len(p.Buckets))
			for i := range p.Buckets {
				values[i] = p.Buckets[i].SurfaceValue
			}

			p.SurfaceInk = inks.NumericInk(p.SurfaceMetric, values, palette.GetPalette(p.SurfacePalette))
		}
	}

	slog.Info("Rendering image", "output", c.Output, "width", c.Width, "height", c.Height)

	return nil
}

// BuildLegendStage builds the legend config from the inks.
func BuildLegendStage(c *stages.CommonState, p *State) error {
	pos, orient := legend.ResolveOptions(
		c.RootConfig.LegendPositionStr(),
		c.RootConfig.LegendOrientationStr(),
	)

	builder := legend.Builder{
		Position: pos, Orientation: orient,
		FillInk: p.Inks.Fill, FillMetric: p.FillMetric,
		BorderInk: p.Inks.Border, BorderMetric: p.BorderMetric,
		SizeMetric: effectiveSizeMetric(p.Size),
	}

	if p.SurfaceMetric != "" && (p.SurfaceMetric != p.FillMetric || p.SurfacePalette != p.FillPalette) {
		builder.AdditionalEntries = append(builder.AdditionalEntries, legend.Entry{
			Role: legend.RoleSurface, MetricName: string(p.SurfaceMetric), Ink: p.SurfaceInk,
		})
	}

	p.LegendConfig = builder.Build()
	if p.LegendConfig != nil {
		for _, bucket := range p.Buckets {
			if len(bucket.Files) == 0 {
				continue
			}

			p.LegendConfig.LabelSample = legend.LabelSample{
				Shape: legend.LabelSampleCircle,
				Lines: buildDiscLabel(bucket, LabelMetrics{
					Size:    effectiveSizeMetric(p.Size),
					Fill:    p.FillMetric,
					Border:  p.BorderMetric,
					Surface: p.SurfaceMetric,
				}),
			}

			break
		}
	}

	return nil
}

// LayoutStage runs the spiral layout algorithm and applies disc sizing.
func LayoutStage(c *stages.CommonState, p *State) error {
	bounds := c.DrawingBounds
	availH := bounds.Height()

	layout := Layout(p.Buckets, c.Width, availH, p.Resolution)
	maxDisc := MaxDiscRadius(len(p.Buckets), c.Width, availH, p.Resolution)

	ApplyDiscSizes(layout.Nodes, p.Buckets, maxDisc)

	if bounds.MinY > 0 {
		dy := float64(bounds.MinY)
		layout.CY += dy

		for i := range layout.Nodes {
			layout.Nodes[i].Y += dy
		}
	}

	p.Layout = layout
	p.DiscLabels = buildDiscLabels(layout.Nodes, p.Buckets, p.Inks.Fill, LabelMetrics{
		Size:    effectiveSizeMetric(p.Size),
		Fill:    p.FillMetric,
		Border:  p.BorderMetric,
		Surface: p.SurfaceMetric,
	})

	return nil
}

// RenderStage renders the spiral to a canvas and attaches the legend.
func RenderStage(c *stages.CommonState, p *State) error {
	var (
		triangles  []surface.Triangle
		surfaceInk inks.Ink
	)

	if p.SurfaceEnabled {
		values := make([]float64, len(p.Buckets))
		for index := range p.Buckets {
			values[index] = p.Buckets[index].SurfaceValue
		}

		triangles = BuildSurface(p.Layout, values, surfaceSeed(p.Layout))
		surfaceInk = p.SurfaceInk

		if len(triangles) == 0 {
			slog.Warn(
				"surface rendering unavailable; rendering spiral without surface",
				"points", len(p.Layout.Nodes),
			)
		}
	}

	cv := RenderToCanvas(p.Layout, p.Buckets, c.Width, c.Height, p.Inks, triangles, surfaceInk)

	legend.RenderInto(cv, p.LegendConfig)

	c.Canvas = cv

	return nil
}

// LogResult logs the final summary line.
func LogResult(c *stages.CommonState, p *State) error {
	files, dirs := stages.CountAll(c.Root)

	slog.Info(
		"Rendered spiral",
		"files", files,
		"directories", dirs,
		"width", c.Width,
		"height", c.Height,
		"size_metric", string(p.Size),
		"fill_metric", string(p.FillMetric),
		"fill_palette", string(p.FillPalette),
		"border_metric", string(p.BorderMetric),
		"border_palette", string(p.BorderPalette),
	)

	return nil
}
