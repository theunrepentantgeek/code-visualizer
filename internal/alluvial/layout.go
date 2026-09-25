package alluvial

import (
	"cmp"
	"math"
	"slices"
)

const (
	layoutHorizontalMargin = 80.0
	layoutVerticalMargin   = 40.0
	layoutBandGap          = 6.0
)

// Layout contains the positioned release columns and the paths joining them.
type Layout struct {
	Columns []ColumnLayout
	Flows   []Flow
	Top     float64
	Bottom  float64
}

// ColumnLayout positions one release column.
type ColumnLayout struct {
	Reference string
	X         float64
	Bands     []Band
}

// Band is the vertical extent allocated to one directory in a release column.
type Band struct {
	Path         string
	Top          float64
	Bottom       float64
	Width        float64
	FillValue    float64
	HasFillValue bool
}

// Flow is a filled quadrilateral joining one directory between adjacent columns.
// A zero-height endpoint represents an introduced or removed directory.
type Flow struct {
	Path                string
	FillValue           float64
	HasFillValue        bool
	FromX, ToX          float64
	FromTop, FromBottom float64
	ToTop, ToBottom     float64
}

// LayoutData positions release columns on a shared metric scale and aligns
// paths across transitions.
func LayoutData(data Data, width, height int) Layout {
	if width <= 0 || height <= 0 {
		return Layout{}
	}

	columns := columnsWithZeroPlaceholders(data.Columns)
	top, bottom := layoutBounds(height)
	layout := Layout{
		Columns: make([]ColumnLayout, len(columns)),
		Top:     top,
		Bottom:  bottom,
	}
	bandsByReference := make(map[string]map[string]Band, len(columns))
	xByReference := make(map[string]float64, len(columns))
	scale := maxColumnTotal(columns)
	maximumBandCount := maxBandCount(columns)
	gap := min(layoutBandGap, (bottom-top)/float64(2*maximumBandCount))
	available := bottom - top - gap*float64(maximumBandCount-1)

	for index, column := range columns {
		x := columnX(index, len(columns), width)

		bands := layoutBands(column.Values, top, bottom-top, scale, available, gap)
		for bandIndex := range bands {
			band := &bands[bandIndex]
			band.FillValue, band.HasFillValue = fillValueFor(data, column.Reference, band.Path)
		}

		layout.Columns[index] = ColumnLayout{
			Reference: column.Reference,
			X:         x,
			Bands:     bands,
		}
		bandsByReference[column.Reference] = bandsByPath(bands)
		xByReference[column.Reference] = x
	}

	for _, transition := range data.Transitions {
		if flow, ok := transitionFlow(transition, bandsByReference, xByReference); ok {
			flow.FillValue, flow.HasFillValue = fillValueFor(data, transition.ToReference, flow.Path)
			layout.Flows = append(layout.Flows, flow)
		}
	}

	return layout
}

func fillValueFor(data Data, reference, directoryPath string) (float64, bool) {
	if data.FillValuesByReference != nil {
		value, ok := data.FillValuesByReference[reference][directoryPath]

		return value, ok
	}

	value, ok := data.FillValues[directoryPath]

	return value, ok
}

func columnsWithZeroPlaceholders(columns []Column) []Column {
	activePaths := make(map[string]struct{})

	for _, column := range columns {
		for _, value := range column.Values {
			if positiveFinite(value.Width) {
				activePaths[value.Path] = struct{}{}
			}
		}
	}

	paths := make([]string, 0, len(activePaths))
	for path := range activePaths {
		paths = append(paths, path)
	}

	slices.Sort(paths)

	result := make([]Column, len(columns))
	for index, column := range columns {
		widths := valuesByPath(column.Values)

		values := make([]Value, 0, len(paths))
		for _, path := range paths {
			values = append(values, Value{Path: path, Width: widths[path]})
		}

		result[index] = Column{Reference: column.Reference, Values: values}
	}

	return result
}

func layoutBounds(height int) (top, bottom float64) {
	margin := min(layoutVerticalMargin, float64(height)/10)

	return margin, float64(height) - margin
}

func transitionFlow(
	transition Transition,
	bandsByReference map[string]map[string]Band,
	xByReference map[string]float64,
) (Flow, bool) {
	from, fromOK := bandsByReference[transition.FromReference][transition.Path]
	to, toOK := bandsByReference[transition.ToReference][transition.Path]
	fromX := xByReference[transition.FromReference]
	toX := xByReference[transition.ToReference]

	switch {
	case fromOK && toOK:
		return flowBetween(transition.Path, fromX, toX, from, to), true
	case !fromOK && toOK && positiveFinite(transition.ToWidth):
		center := (to.Top + to.Bottom) / 2

		return Flow{
			Path: transition.Path, FromX: fromX, ToX: toX,
			FromTop: center, FromBottom: center, ToTop: to.Top, ToBottom: to.Bottom,
		}, true
	case fromOK && !toOK && positiveFinite(transition.FromWidth):
		center := (from.Top + from.Bottom) / 2

		return Flow{
			Path: transition.Path, FromX: fromX, ToX: toX,
			FromTop: from.Top, FromBottom: from.Bottom, ToTop: center, ToBottom: center,
		}, true
	default:
		return Flow{}, false
	}
}

func columnX(index, count, width int) float64 {
	margin := min(layoutHorizontalMargin, float64(width)/10)
	if count <= 1 {
		return float64(width) / 2
	}

	return margin + float64(index)*(float64(width)-2*margin)/float64(count-1)
}

func maxColumnTotal(columns []Column) float64 {
	maximum := 0.0

	for _, column := range columns {
		total := 0.0

		for _, value := range column.Values {
			if positiveFinite(value.Width) {
				total += value.Width
			}
		}

		maximum = max(maximum, total)
	}

	return maximum
}

func maxBandCount(columns []Column) int {
	count := 0

	for _, column := range columns {
		count = max(count, len(column.Values))
	}

	return count
}

func layoutBands(values []Value, top, verticalSpace, scale, available, gap float64) []Band {
	values = append([]Value(nil), values...)
	slices.SortFunc(values, func(left, right Value) int {
		return cmp.Compare(left.Path, right.Path)
	})

	total := 0.0

	for _, value := range values {
		if positiveFinite(value.Width) {
			total += value.Width
		}
	}

	if scale == 0 || len(values) == 0 || available <= 0 {
		return nil
	}

	bands := make([]Band, 0, len(values))
	stackHeight := available*total/scale + gap*float64(len(values)-1)
	y := top + (verticalSpace-stackHeight)/2

	for index, value := range values {
		bandHeight := available * value.Width / scale

		if index > 0 {
			y += gap
		}

		bands = append(bands, Band{Path: value.Path, Top: y, Bottom: y + bandHeight, Width: value.Width})
		y += bandHeight
	}

	return bands
}

func bandsByPath(bands []Band) map[string]Band {
	result := make(map[string]Band, len(bands))
	for _, band := range bands {
		result[band.Path] = band
	}

	return result
}

func flowBetween(path string, fromX, toX float64, from, to Band) Flow {
	return Flow{
		Path: path, FromX: fromX, ToX: toX,
		FromTop: from.Top, FromBottom: from.Bottom,
		ToTop: to.Top, ToBottom: to.Bottom,
	}
}

func positiveFinite(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
