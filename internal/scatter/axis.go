package scatter

import "github.com/theunrepentantgeek/code-visualizer/internal/metric"

// AxisSpec identifies the metric and kind used for one scatter axis.
type AxisSpec struct {
	Metric metric.Name
	Kind   metric.Kind
	Scale  ScaleType
}

// AxisValue carries one file's resolved value for a scatter axis.
type AxisValue struct {
	Numeric  float64
	Category string
}

// PlotRect is the drawable chart area after reserving margins.
type PlotRect struct {
	X float64
	Y float64
	W float64
	H float64
}

// AxisTick is a labeled numeric tick at an absolute canvas position.
type AxisTick struct {
	Value    float64
	Label    string
	Position float64
}

// AxisBand is a categorical swimlane spanning absolute canvas positions.
type AxisBand struct {
	Label  string
	Start  float64
	End    float64
	Center float64
}
