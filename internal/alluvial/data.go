package alluvial

import (
	"cmp"
	"slices"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// Data is the deterministic, renderer-independent alluvial input model.
type Data struct {
	Columns               []Column
	Transitions           []Transition
	FillValues            map[string]float64
	FillValuesByReference map[string]map[string]float64
	MutedPaths            map[string]struct{}
}

// FillValuesForInk returns every displayed numeric fill value, including
// values for the same path at different destination snapshots.
func (d Data) FillValuesForInk() []float64 {
	if d.FillValuesByReference == nil {
		values := make([]float64, 0, len(d.FillValues))

		for _, value := range d.FillValues {
			values = append(values, value)
		}

		return values
	}

	values := make([]float64, 0)

	for _, column := range d.Columns {
		for _, value := range column.Values {
			fillValue, ok := d.FillValuesByReference[column.Reference][value.Path]
			if ok {
				values = append(values, fillValue)
			}
		}
	}

	return values
}

// Column contains metric widths for a single reference snapshot.
type Column struct {
	Reference string
	Values    []Value
}

// Value identifies a directory and its metric width in one snapshot.
type Value struct {
	Path  string
	Width float64
}

// Transition joins one path in adjacent reference columns. A missing endpoint
// has zero width, allowing a renderer to taper introduced and removed paths.
type Transition struct {
	FromReference string
	ToReference   string
	Path          string
	FromWidth     float64
	ToWidth       float64
}

// BuildData creates ordered snapshot columns and deterministic transitions.
// Widths always come from the reference snapshot itself, never from a delta.
func BuildData(snapshots []Snapshot, options Options) (Data, error) {
	if options.Metric == "" {
		return Data{}, eris.New("alluvial width metric is required")
	}

	data := Data{Columns: make([]Column, 0, len(snapshots))}

	fillMetric := options.FillMetric
	if fillMetric == "" {
		fillMetric = options.Metric
	}

	fillSnapshots := make([]map[string]float64, 0, len(snapshots))
	hasBands := false

	for _, snapshot := range snapshots {
		if snapshot.Bands == nil {
			return Data{}, eris.Errorf("alluvial reference %q has no evaluated bands", snapshot.Reference)
		}

		values := selectedValues(snapshot.Bands, options.Metric)
		if len(values) > 0 {
			hasBands = true
		}

		fillSnapshots = append(
			fillSnapshots,
			selectedMetricValues(snapshot.Bands, fillMetric),
		)
		data.Columns = append(data.Columns, Column{
			Reference: snapshot.Reference,
			Values:    values,
		})
	}

	if !hasBands {
		return Data{}, eris.New("alluvial target contains no files in any reference")
	}

	data.Columns, data.MutedPaths = applyConstantBands(data.Columns, options.ConstantBands)
	data.Transitions = buildTransitions(data.Columns)

	data.FillValues, data.FillValuesByReference = buildFillValues(
		data.Columns, fillSnapshots, options.FillTemporal,
	)
	removeMutedFillValues(&data)

	return data, nil
}

func removeMutedFillValues(data *Data) {
	for directoryPath := range data.MutedPaths {
		delete(data.FillValues, directoryPath)

		for _, values := range data.FillValuesByReference {
			delete(values, directoryPath)
		}
	}
}

func selectedValues(
	bands map[string]*model.MetricContainer,
	metricName metric.Name,
) []Value {
	values := make([]Value, 0, len(bands))
	for directoryPath, band := range bands {
		width := float64(0)
		if band != nil {
			width = metricValue(*band, metricName)
		}

		values = append(values, Value{
			Path:  directoryPath,
			Width: width,
		})
	}

	slices.SortFunc(values, func(left, right Value) int {
		return cmp.Compare(left.Path, right.Path)
	})

	return values
}

func metricValue(values model.MetricContainer, metricName metric.Name) float64 {
	if value, ok := values.Quantity(metricName); ok {
		return float64(value)
	}

	if value, ok := values.Measure(metricName); ok {
		return value
	}

	return 0
}

func selectedMetricValues(
	bands map[string]*model.MetricContainer,
	metricName metric.Name,
) map[string]float64 {
	values := make(map[string]float64, len(bands))
	for directoryPath, band := range bands {
		if band == nil {
			values[directoryPath] = 0

			continue
		}

		values[directoryPath] = metricValue(*band, metricName)
	}

	return values
}

func buildFillValues(
	columns []Column,
	snapshots []map[string]float64,
	temporal metric.TemporalName,
) (map[string]float64, map[string]map[string]float64) {
	values := make(map[string]float64)

	if len(snapshots) == 0 {
		return values, nil
	}

	last := snapshots[len(snapshots)-1]

	if temporal == metric.TemporalStepDelta {
		return buildStepDeltaFillValues(columns, snapshots)
	}

	for _, column := range columns {
		for _, value := range column.Values {
			fillValue := temporalFillValue(temporal, last, snapshots, value.Path)

			values[value.Path] = fillValue
		}
	}

	return values, nil
}

func buildStepDeltaFillValues(
	columns []Column,
	snapshots []map[string]float64,
) (map[string]float64, map[string]map[string]float64) {
	values := make(map[string]float64)
	byReference := make(map[string]map[string]float64, len(columns))

	for index, column := range columns {
		byPath := make(map[string]float64)

		if index == 0 {
			byReference[column.Reference] = byPath

			continue
		}

		for _, value := range column.Values {
			fillValue := snapshots[index][value.Path] - snapshots[index-1][value.Path]
			values[value.Path] = fillValue
			byPath[value.Path] = fillValue
		}

		byReference[column.Reference] = byPath
	}

	return values, byReference
}

func temporalFillValue(
	temporal metric.TemporalName,
	last map[string]float64,
	snapshots []map[string]float64,
	directoryPath string,
) float64 {
	switch temporal {
	case metric.TemporalDelta:
		return last[directoryPath] - snapshots[0][directoryPath]
	default:
		return last[directoryPath]
	}
}

func buildTransitions(columns []Column) []Transition {
	transitions := make([]Transition, 0)

	for i := 0; i+1 < len(columns); i++ {
		from, to := columns[i], columns[i+1]
		fromValues := valuesByPath(from.Values)
		toValues := valuesByPath(to.Values)
		paths := unionPaths(fromValues, toValues)

		for _, directoryPath := range paths {
			transitions = append(transitions, Transition{
				FromReference: from.Reference,
				ToReference:   to.Reference,
				Path:          directoryPath,
				FromWidth:     fromValues[directoryPath],
				ToWidth:       toValues[directoryPath],
			})
		}
	}

	return transitions
}

func valuesByPath(values []Value) map[string]float64 {
	result := make(map[string]float64, len(values))
	for _, value := range values {
		result[value.Path] = value.Width
	}

	return result
}

func unionPaths(left, right map[string]float64) []string {
	paths := make([]string, 0, len(left)+len(right))

	seen := make(map[string]struct{}, len(left)+len(right))
	for directoryPath := range left {
		seen[directoryPath] = struct{}{}
		paths = append(paths, directoryPath)
	}

	for directoryPath := range right {
		if _, ok := seen[directoryPath]; !ok {
			paths = append(paths, directoryPath)
		}
	}

	slices.Sort(paths)

	return paths
}
