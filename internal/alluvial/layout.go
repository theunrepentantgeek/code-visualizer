package alluvial

import (
	"math"
	"sort"
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
	Path   string
	Top    float64
	Bottom float64
}

// Flow is a filled quadrilateral joining one directory between adjacent columns.
// A zero-height endpoint represents an introduced or removed directory.
type Flow struct {
	Path                string
	FromX, ToX          float64
	FromTop, FromBottom float64
	ToTop, ToBottom     float64
}

// LayoutData positions each release column independently, preserving metric
// proportions within that snapshot and aligning paths across transitions.
func LayoutData(data Data, width, height int) Layout {
	if width <= 0 || height <= 0 {
		return Layout{}
	}

	top, bottom := layoutBounds(height)
	layout := Layout{
		Columns: make([]ColumnLayout, len(data.Columns)),
		Top:     top,
		Bottom:  bottom,
	}
	bandsByReference := make(map[string]map[string]Band, len(data.Columns))

	for index, column := range data.Columns {
		x := columnX(index, len(data.Columns), width)
		bands := layoutBands(column.Values, top, bottom)
		layout.Columns[index] = ColumnLayout{
			Reference: column.Reference,
			X:         x,
			Bands:     bands,
		}
		bandsByReference[column.Reference] = bandsByPath(bands)
	}

	for _, transition := range data.Transitions {
		from, fromOK := bandsByReference[transition.FromReference][transition.Path]
		to, toOK := bandsByReference[transition.ToReference][transition.Path]

		switch {
		case fromOK && toOK:
			layout.Flows = append(layout.Flows, flowBetween(
				transition.Path,
				columnXForReference(layout.Columns, transition.FromReference),
				columnXForReference(layout.Columns, transition.ToReference),
				from,
				to,
			))
		case !fromOK && toOK && positiveFinite(transition.ToWidth):
			center := (to.Top + to.Bottom) / 2
			layout.Flows = append(layout.Flows, Flow{
				Path: transition.Path, FromX: columnXForReference(layout.Columns, transition.FromReference),
				ToX:     columnXForReference(layout.Columns, transition.ToReference),
				FromTop: center, FromBottom: center, ToTop: to.Top, ToBottom: to.Bottom,
			})
		case fromOK && !toOK && positiveFinite(transition.FromWidth):
			center := (from.Top + from.Bottom) / 2
			layout.Flows = append(layout.Flows, Flow{
				Path: transition.Path, FromX: columnXForReference(layout.Columns, transition.FromReference),
				ToX:     columnXForReference(layout.Columns, transition.ToReference),
				FromTop: from.Top, FromBottom: from.Bottom, ToTop: center, ToBottom: center,
			})
		}
	}

	return layout
}

func layoutBounds(height int) (float64, float64) {
	margin := min(layoutVerticalMargin, float64(height)/10)

	return margin, float64(height) - margin
}

func columnX(index, count, width int) float64 {
	margin := min(layoutHorizontalMargin, float64(width)/10)
	if count <= 1 {
		return float64(width) / 2
	}

	return margin + float64(index)*(float64(width)-2*margin)/float64(count-1)
}

func layoutBands(values []Value, top, bottom float64) []Band {
	values = append([]Value(nil), values...)
	sort.Slice(values, func(i, j int) bool {
		return values[i].Path < values[j].Path
	})

	total := 0.0
	filtered := values[:0]
	for _, value := range values {
		if !positiveFinite(value.Width) {
			continue
		}

		total += value.Width
		filtered = append(filtered, value)
	}
	if total == 0 || bottom <= top {
		return nil
	}

	gap := min(layoutBandGap, (bottom-top)/float64(2*len(filtered)))
	available := bottom - top - gap*float64(len(filtered)-1)
	bands := make([]Band, 0, len(filtered))
	y := top
	for _, value := range filtered {
		bandHeight := available * value.Width / total
		bands = append(bands, Band{Path: value.Path, Top: y, Bottom: y + bandHeight})
		y += bandHeight + gap
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

func columnXForReference(columns []ColumnLayout, reference string) float64 {
	for _, column := range columns {
		if column.Reference == reference {
			return column.X
		}
	}

	return 0
}

func positiveFinite(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
