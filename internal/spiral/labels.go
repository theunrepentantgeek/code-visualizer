package spiral

import (
	"strconv"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

const discLabelPadding = 2.0

// LabelMetrics identifies the metrics included in each disc label.
type LabelMetrics struct {
	Size      metric.Name
	Fill      metric.Name
	Border    metric.Name
	Surface   metric.Name
	Requested stages.RequestedMetrics
}

type numericLabelValue struct {
	value     float64
	available bool
}

func effectiveSizeMetric(name metric.Name) metric.Name {
	if name == "" {
		return commitCountMetric
	}

	return name
}

func buildDiscLabel(bucket TimeBucket, metrics LabelMetrics) []string {
	metrics.Size = effectiveSizeMetric(metrics.Size)

	lines := make([]string, 0, 6)
	lines = append(
		lines,
		strconv.Itoa(bucket.Start.Day()),
		bucket.Start.Format("Jan"),
	)
	seen := make(map[metric.Name]bool, 4)

	for _, role := range []struct {
		name metric.Name
		role labelRole
	}{
		{metrics.Size, labelSize},
		{metrics.Fill, labelFill},
		{metrics.Border, labelBorder},
		{metrics.Surface, labelSurface},
	} {
		if role.name == "" || seen[role.name] {
			continue
		}

		seen[role.name] = true
		if value, ok := discMetricValue(bucket, role.name, role.role, metrics.Requested); ok {
			lines = append(lines, value)
		}
	}

	return lines
}

func buildLegendLabelSample(metrics LabelMetrics) []string {
	metrics.Size = effectiveSizeMetric(metrics.Size)
	lines := make([]string, 0, 6)
	lines = append(lines, "Day", "Month")
	seen := make(map[metric.Name]bool, 4)

	for _, name := range []metric.Name{metrics.Size, metrics.Fill, metrics.Border, metrics.Surface} {
		if name == "" || seen[name] {
			continue
		}

		seen[name] = true
		lines = append(lines, string(name))
	}

	return lines
}

type labelRole int

const (
	labelSize labelRole = iota
	labelFill
	labelBorder
	labelSurface
)

func discMetricValue(
	bucket TimeBucket,
	name metric.Name,
	role labelRole,
	requested stages.RequestedMetrics,
) (string, bool) {
	switch role {
	case labelSize:
		if name == commitCountMetric {
			return strconv.FormatFloat(float64(len(bucket.Files)), 'f', -1, 64), true
		}

		if !bucket.SizeValueAvailable {
			return "", false
		}

		return strconv.FormatFloat(bucket.SizeValue, 'f', -1, 64), true
	case labelFill:
		return colourLabelValue(
			numericLabelValue{value: bucket.FillValue, available: bucket.FillValueAvailable},
			bucket.FillLabel, name, requested,
		)
	case labelBorder:
		return colourLabelValue(
			numericLabelValue{value: bucket.BorderValue, available: bucket.BorderValueAvailable},
			bucket.BorderLabel, name, requested,
		)
	default:
		if !bucket.SurfaceValueAvailable {
			return "", false
		}

		return strconv.FormatFloat(bucket.SurfaceValue, 'f', -1, 64), true
	}
}

func colourLabelValue(
	value numericLabelValue,
	label string,
	name metric.Name,
	requested stages.RequestedMetrics,
) (string, bool) {
	if label != "" {
		return label, true
	}

	descriptor, ok := requested.DescriptorFor(name)
	if !ok || descriptor.Kind == metric.Classification {
		return "", false
	}

	if !value.available {
		return "", false
	}

	return strconv.FormatFloat(value.value, 'f', -1, 64), true
}

func buildDiscLabels(
	nodes []SpiralNode,
	buckets []TimeBucket,
	fillInk inks.Ink,
	metrics LabelMetrics,
) []canvas.BlockLabel {
	count := min(len(nodes), len(buckets))
	labels := make([]canvas.BlockLabel, 0, count)
	nodes = nodes[:count]
	buckets = buckets[:count]

	for i, node := range nodes {
		if node.DiscRadius <= 0 {
			continue
		}

		size := 2 * (node.DiscRadius - discLabelPadding)
		if size <= 0 {
			continue
		}

		bucket := buckets[i]
		fill := fillInk.Dip(metricValue(bucket.FillValue, bucket.FillLabel, fillInk))
		labels = append(labels, canvas.BlockLabel{
			X:            node.Position.X - node.DiscRadius + discLabelPadding,
			Y:            node.Position.Y - node.DiscRadius + discLabelPadding,
			W:            size,
			H:            size,
			Lines:        buildDiscLabel(bucket, metrics),
			Ink:          canvas.TextColourFor(fill),
			PreserveText: true,
		})
	}

	return labels
}
