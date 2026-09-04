// Package legend constructs and renders legend overlays for visualizations.
// It holds the legend configuration types, computes reservation/layout
// geometry, and decomposes the legend into canvas primitives.
package legend

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/legendlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
)

// Role identifies what visual property a legend entry describes.
type Role string

const (
	RoleFill    Role = "Fill"
	RoleBorder  Role = "Border"
	RoleSize    Role = "Size"
	RoleSurface Role = "Surface"
)

// Entry describes one metric shown in the legend.
type Entry struct {
	Role       Role
	MetricName string
	Ink        inks.Ink
}

// LabelSampleShape specifies the backing shape for a label sample.
type LabelSampleShape int

const (
	// LabelSampleSquare renders the label sample in a square.
	LabelSampleSquare LabelSampleShape = iota
	// LabelSampleCircle renders the label sample in a circle.
	LabelSampleCircle
	// LabelSampleArc renders the label sample in a curved annular segment.
	LabelSampleArc
)

// LabelSample describes the inline label preview displayed before legend entries.
type LabelSample struct {
	Shape LabelSampleShape
	Lines []string
}

// Config holds everything needed to render a legend.
type Config struct {
	Position    model.LegendPosition
	Orientation model.LegendOrientation
	LabelSample LabelSample
	Entries     []Entry
}

// DefaultOrientation returns the default orientation for a given position.
// Top-center and bottom-center default to horizontal; all others to vertical.
func DefaultOrientation(pos model.LegendPosition) model.LegendOrientation {
	switch pos {
	case model.LegendPositionTopCenter, model.LegendPositionBottomCenter:
		return model.LegendOrientationHorizontal
	default:
		return model.LegendOrientationVertical
	}
}

// ReserveSpace computes the width and height reductions needed to reserve
// space for the legend within the canvas. Returns a zero size if the legend
// is disabled or has no entries.
func (cfg *Config) ReserveSpace() geometry.Size {
	data := cfg.toLegendData()

	return legendlayout.ReserveSpace(data, legendlayout.NewBasicMeasurer())
}

// toLegendData converts the legend Config to the backend-facing LegendData.
// Returns nil if the legend is disabled or has no entries.
func (cfg *Config) toLegendData() *model.LegendData {
	if cfg == nil || cfg.Position == model.LegendPositionNone || len(cfg.Entries) == 0 {
		return nil
	}

	entries := make([]model.LegendEntryData, len(cfg.Entries))

	for i, e := range cfg.Entries {
		kind, swatches := inks.LegendData(e.Ink)
		entries[i] = model.LegendEntryData{
			Label:    string(e.Role),
			Metric:   e.MetricName,
			Kind:     kind,
			Swatches: swatches,
			IsBorder: e.Role == RoleBorder,
		}
	}

	orient := cfg.Orientation
	if orient == "" {
		orient = DefaultOrientation(cfg.Position)
	}

	return &model.LegendData{
		Position:    cfg.Position,
		Orientation: orient,
		LabelSample: labelSampleData(cfg.LabelSample),
		Entries:     entries,
	}
}

func labelSampleData(labelSample LabelSample) *model.LegendLabelSample {
	if len(labelSample.Lines) == 0 {
		return nil
	}

	sample := make([]string, 0, len(labelSample.Lines))
	for _, line := range labelSample.Lines {
		if line == "" {
			continue
		}

		sample = append(sample, line)
	}

	if len(sample) == 0 {
		return nil
	}

	return &model.LegendLabelSample{
		Shape: model.LegendLabelSampleShape(labelSample.Shape),
		Lines: sample,
	}
}
