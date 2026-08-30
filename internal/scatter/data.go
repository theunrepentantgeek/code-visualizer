package scatter

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

type metricSource interface {
	Quantity(name metric.Name) (int64, bool)
	Measure(name metric.Name) (float64, bool)
	Classification(name metric.Name) (string, bool)
}

// PointDatum holds the resolved metric values for one plottable node.
type PointDatum struct {
	File      *model.File
	Directory *model.Directory
	X         AxisValue
	Y         AxisValue
	Size      float64
}

// Name returns the display name of the point's file or directory.
func (p PointDatum) Name() string {
	if p.Directory != nil {
		return p.Directory.Name
	}

	if p.File != nil {
		return p.File.Name
	}

	return ""
}

func (p PointDatum) metricContainer() *model.MetricContainer {
	if p.Directory != nil {
		return &p.Directory.MetricContainer
	}

	if p.File != nil {
		return &p.File.MetricContainer
	}

	return nil
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

func (d Dataset) metricSources() []metricSource {
	sources := make([]metricSource, 0, len(d.Points))
	for _, point := range d.Points {
		if point.Directory != nil {
			sources = append(sources, point.Directory)
		} else if point.File != nil {
			sources = append(sources, point.File)
		}
	}

	return sources
}

// CollectDataset walks nodes at the selected grain and keeps those with X, Y, and size values.
func CollectDataset(
	root *model.Directory,
	grain viz.Grain,
	xAxis, yAxis AxisSpec,
	sizeMetric metric.Name,
) Dataset {
	dataset := Dataset{}
	if root == nil {
		return dataset
	}

	dataset.Points = make([]PointDatum, 0, datasetCapacity(root, grain))

	if grain == viz.GrainDirectory {
		model.WalkDirectories(root, func(dir *model.Directory) {
			collectPoint(&dataset, PointDatum{Directory: dir}, xAxis, yAxis, sizeMetric)
		})
	} else {
		model.WalkFiles(root, func(file *model.File) {
			collectPoint(&dataset, PointDatum{File: file}, xAxis, yAxis, sizeMetric)
		})
	}

	return dataset
}

func datasetCapacity(root *model.Directory, grain viz.Grain) int {
	if grain == viz.GrainDirectory {
		return root.AllDirCount + 1
	}

	return root.AllFileCount
}

func collectPoint(d *Dataset, point PointDatum, xAxis, yAxis AxisSpec, sizeMetric metric.Name) {
	container := point.metricContainer()
	x, okX := axisValueForContainer(container, xAxis)
	y, okY := axisValueForContainer(container, yAxis)
	size, okSize := numericValueForContainer(container, sizeMetric)

	if !okX {
		d.Skipped.MissingX++
	}

	if !okY {
		d.Skipped.MissingY++
	}

	if !okSize {
		d.Skipped.MissingSize++
	}

	if !okX || !okY || !okSize {
		return
	}

	point.X = x
	point.Y = y
	point.Size = size
	d.Points = append(d.Points, point)
}

func axisValueForContainer(container *model.MetricContainer, axis AxisSpec) (AxisValue, bool) {
	if container == nil {
		return AxisValue{}, false
	}

	switch axis.Kind {
	case metric.Classification:
		if value, ok := container.Classification(axis.Metric); ok {
			return AxisValue{Category: value}, true
		}
	default:
		if value, ok := container.Quantity(axis.Metric); ok {
			return AxisValue{Numeric: float64(value)}, true
		}

		if value, ok := container.Measure(axis.Metric); ok {
			return AxisValue{Numeric: value}, true
		}
	}

	return AxisValue{}, false
}

func numericValueForContainer(container *model.MetricContainer, name metric.Name) (float64, bool) {
	if container == nil {
		return 0, false
	}

	if value, ok := container.Quantity(name); ok {
		return float64(value), true
	}

	if value, ok := container.Measure(name); ok {
		return value, true
	}

	return 0, false
}
