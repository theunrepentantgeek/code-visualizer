package treemap

import (
	"strconv"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	pkginks "github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

const blockLabelPadding = 4.0

// LabelMetrics identifies which metric values should appear in each file label.
type LabelMetrics struct {
	Size   metric.Name
	Fill   metric.Name
	Border metric.Name
}

func buildBlockLabels(
	rect TreemapRectangle,
	dir *model.Directory,
	fillInk canvas.Ink,
	metrics LabelMetrics,
) []canvas.BlockLabel {
	labels := make([]canvas.BlockLabel, 0)
	if dir == nil || !rect.IsDirectory {
		return labels
	}

	fileIdx := 0
	dirIdx := 0

	for i := range rect.Children {
		child := rect.Children[i]
		if child.IsDirectory {
			labels, dirIdx = appendDirectoryLabels(labels, child, dir, dirIdx, fillInk, metrics)

			continue
		}

		labels, fileIdx = appendFileLabels(labels, child, dir, fileIdx, fillInk, metrics)
	}

	return labels
}

func appendDirectoryLabels(
	labels []canvas.BlockLabel,
	child TreemapRectangle,
	dir *model.Directory,
	dirIdx int,
	fillInk canvas.Ink,
	metrics LabelMetrics,
) ([]canvas.BlockLabel, int) {
	if dirIdx >= len(dir.Dirs) {
		return labels, dirIdx
	}

	labels = append(labels, buildBlockLabels(child, dir.Dirs[dirIdx], fillInk, metrics)...)

	return labels, dirIdx + 1
}

func appendFileLabels(
	labels []canvas.BlockLabel,
	child TreemapRectangle,
	dir *model.Directory,
	fileIdx int,
	fillInk canvas.Ink,
	metrics LabelMetrics,
) ([]canvas.BlockLabel, int) {
	if fileIdx >= len(dir.Files) {
		return labels, fileIdx
	}

	file := dir.Files[fileIdx]
	if label, ok := buildFileLabel(child, file, fillInk, metrics); ok {
		labels = append(labels, label)
	}

	return labels, fileIdx + 1
}

func buildFileLabel(
	rect TreemapRectangle,
	file *model.File,
	fillInk canvas.Ink,
	metrics LabelMetrics,
) (canvas.BlockLabel, bool) {
	if file == nil {
		return canvas.BlockLabel{}, false
	}

	bounds, ok := insetLabelBounds(rect, blockLabelPadding)
	if !ok {
		return canvas.BlockLabel{}, false
	}

	lines := []string{rect.Label}
	lines = appendMetricLine(lines, file, metrics.Size)
	lines = appendMetricLine(lines, file, metrics.Fill)
	lines = appendMetricLine(lines, file, metrics.Border)

	fillColour := fillInk.Dip(pkginks.MetricValueForFile(file, fillInk))

	return canvas.BlockLabel{
		X:     bounds.x,
		Y:     bounds.y,
		W:     bounds.w,
		H:     bounds.h,
		Lines: lines,
		Ink:   canvas.TextColourFor(fillColour),
	}, true
}

type labelBounds struct {
	x float64
	y float64
	w float64
	h float64
}

func insetLabelBounds(rect TreemapRectangle, padding float64) (labelBounds, bool) {
	bounds := labelBounds{
		x: rect.X + padding,
		y: rect.Y + padding,
		w: rect.W - 2*padding,
		h: rect.H - 2*padding,
	}
	if bounds.w <= 0 || bounds.h <= 0 {
		return labelBounds{}, false
	}

	return bounds, true
}

func appendMetricLine(lines []string, file *model.File, name metric.Name) []string {
	if line, ok := formatMetricValue(file, name); ok {
		return append(lines, line)
	}

	return lines
}

func formatMetricValue(file *model.File, name metric.Name) (string, bool) {
	if name == "" || file == nil {
		return "", false
	}

	if value, ok := file.Quantity(name); ok {
		return strconv.FormatInt(value, 10), true
	}

	if value, ok := file.Measure(name); ok {
		return strconv.FormatFloat(value, 'f', -1, 64), true
	}

	if value, ok := file.Classification(name); ok {
		return value, true
	}

	return "", false
}
