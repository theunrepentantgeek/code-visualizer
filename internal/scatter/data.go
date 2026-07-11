package scatter

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// PointDatum holds the resolved metric values for one plottable file.
type PointDatum struct {
	File *model.File
	X    AxisValue
	Y    AxisValue
	Size float64
}

// SkipCounts records how many files were excluded for missing required values.
type SkipCounts struct {
	MissingX    int
	MissingY    int
	MissingSize int
}

// Total returns the total number of files skipped for any reason.
// Note: a single file may be counted in multiple fields if it is missing
// more than one required value, so Total may exceed the number of distinct
// skipped files.
func (s SkipCounts) Total() int {
	return s.MissingX + s.MissingY + s.MissingSize
}

// Dataset is the subset of files that can be plotted, plus skip statistics.
type Dataset struct {
	Points  []PointDatum
	Skipped SkipCounts
}

// Files returns the plotted files in dataset order.
func (d Dataset) Files() []*model.File {
	files := make([]*model.File, 0, len(d.Points))
	for _, point := range d.Points {
		files = append(files, point.File)
	}

	return files
}

// CollectDataset walks the tree and keeps only files with X, Y, and size values.
func CollectDataset(root *model.Directory, xAxis, yAxis AxisSpec, sizeMetric metric.Name) Dataset {
	dataset := Dataset{}
	if root == nil {
		return dataset
	}

	model.WalkFiles(root, func(file *model.File) {
		x, okX := axisValueForFile(file, xAxis)
		y, okY := axisValueForFile(file, yAxis)
		size, okSize := numericValueForFile(file, sizeMetric)

		if !okX {
			dataset.Skipped.MissingX++
		}

		if !okY {
			dataset.Skipped.MissingY++
		}

		if !okSize {
			dataset.Skipped.MissingSize++
		}

		if !okX || !okY || !okSize {
			return
		}

		dataset.Points = append(dataset.Points, PointDatum{
			File: file,
			X:    x,
			Y:    y,
			Size: size,
		})
	})

	return dataset
}

func axisValueForFile(file *model.File, axis AxisSpec) (AxisValue, bool) {
	switch axis.Kind {
	case metric.Classification:
		if value, ok := file.Classification(axis.Metric); ok {
			return AxisValue{Category: value}, true
		}
	default:
		if value, ok := file.Quantity(axis.Metric); ok {
			return AxisValue{Numeric: float64(value)}, true
		}

		if value, ok := file.Measure(axis.Metric); ok {
			return AxisValue{Numeric: value}, true
		}
	}

	return AxisValue{}, false
}

func numericValueForFile(file *model.File, name metric.Name) (float64, bool) {
	if value, ok := file.Quantity(name); ok {
		return float64(value), true
	}

	if value, ok := file.Measure(name); ok {
		return value, true
	}

	return 0, false
}
