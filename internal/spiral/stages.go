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
)

// ResolveMetrics resolves metric, resolution, and label settings from the
// spiral config and populates c.Requested.
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
	p.Labels = resolveLabels(cfg)

	c.Requested = collectRequestedMetrics(p.Size, cfg.Fill, cfg.Border, cfg.SurfaceMetric)

	return nil
}

func resolveResolution(cfg *config.Spiral) Resolution {
	if r := stages.PtrString(cfg.Resolution); r == "hourly" {
		return Hourly
	}

	return Daily
}

func resolveLabels(cfg *config.Spiral) LabelMode {
	if lbl := stages.PtrString(cfg.Labels); lbl != "" {
		return LabelMode(lbl)
	}

	return LabelLaps
}

// collectRequestedMetrics merges size, fill, border, and surface into a
// deduplicated metric set. When size is empty (spiral defaults to commit
// count), only configured colour and surface metrics contribute.
func collectRequestedMetrics(
	size metric.Name,
	fill, border, surface *config.MetricSpec,
) stages.RequestedMetrics {
	seen := map[metric.Name]bool{}
	names := make([]metric.Name, 0, 4)

	if size != "" {
		seen[size] = true
		names = append(names, size)
	}

	for _, spec := range []*config.MetricSpec{fill, border, surface} {
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

	entries := make([]legend.Entry, 0, 4)
	if p.FillMetric != "" {
		entries = append(entries, legend.Entry{
			Role: legend.RoleFill, MetricName: string(p.FillMetric), Ink: p.Inks.Fill,
		})
	}
	if p.BorderMetric != "" {
		entries = append(entries, legend.Entry{
			Role: legend.RoleBorder, MetricName: string(p.BorderMetric), Ink: p.Inks.Border,
		})
	}
	if p.Size != "" && p.Size != p.FillMetric {
		entries = append(entries, legend.Entry{
			Role: legend.RoleSize, MetricName: string(p.Size), Ink: inks.FixedInk(palette.White),
		})
	}
	if p.SurfaceMetric != "" && (p.SurfaceMetric != p.FillMetric || p.SurfacePalette != p.FillPalette) {
		entries = append(entries, legend.Entry{
			Role: legend.RoleSurface, MetricName: string(p.SurfaceMetric), Ink: p.SurfaceInk,
		})
	}

	p.LegendConfig = legend.Build(pos, orient, entries)

	return nil
}

// LayoutStage runs the spiral layout algorithm and applies disc sizing.
func LayoutStage(c *stages.CommonState, p *State) error {
	bounds := c.DrawingBounds
	availH := bounds.Height()

	layout := Layout(p.Buckets, c.Width, availH, p.Resolution, p.Labels)
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

	return nil
}

// RenderStage renders the spiral to a canvas and attaches the legend.
func RenderStage(c *stages.CommonState, p *State) error {
	cv := RenderToCanvas(p.Layout, p.Buckets, c.Width, c.Height, p.Inks)

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
