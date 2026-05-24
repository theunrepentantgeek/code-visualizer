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
	if dir == nil {
		return nil
	}

	labels := make([]canvas.BlockLabel, 0)
	if !rect.IsDirectory {
		return labels
	}

	fileIdx := 0
	dirIdx := 0
	for i := range rect.Children {
		child := rect.Children[i]
		if child.IsDirectory && dirIdx < len(dir.Dirs) {
			labels = append(labels, buildBlockLabels(child, dir.Dirs[dirIdx], fillInk, metrics)...)
			dirIdx++
			continue
		}

		if child.IsDirectory || fileIdx >= len(dir.Files) {
			continue
		}

		if label, ok := buildFileLabel(child, dir.Files[fileIdx], fillInk, metrics); ok {
			labels = append(labels, label)
		}
		fileIdx++
	}

	return labels
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

	x, y, w, h, ok := insetLabelBounds(rect, blockLabelPadding)
	if !ok {
		return canvas.BlockLabel{}, false
	}

	lines := []string{rect.Label}
	if line, ok := formatMetricValue(file, metrics.Size); ok {
		lines = append(lines, line)
	}
	if metrics.Fill != "" {
		if line, ok := formatMetricValue(file, metrics.Fill); ok {
			lines = append(lines, line)
		}
	}
	if metrics.Border != "" {
		if line, ok := formatMetricValue(file, metrics.Border); ok {
			lines = append(lines, line)
		}
	}

	fillColour := fillInk.Dip(pkginks.MetricValueForFile(file, fillInk))

	return canvas.BlockLabel{
		X:     x,
		Y:     y,
		W:     w,
		H:     h,
		Lines: lines,
		Ink:   canvas.TextColourFor(fillColour),
	}, true
}

func insetLabelBounds(rect TreemapRectangle, padding float64) (x, y, w, h float64, ok bool) {
	w = rect.W - 2*padding
	h = rect.H - 2*padding
	if w <= 0 || h <= 0 {
		return 0, 0, 0, 0, false
	}

	return rect.X + padding, rect.Y + padding, w, h, true
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
