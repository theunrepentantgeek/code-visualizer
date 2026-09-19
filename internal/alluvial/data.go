package alluvial

import (
	"cmp"
	"path"
	"slices"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// Data is the deterministic, renderer-independent alluvial input model.
type Data struct {
	Columns     []Column
	Transitions []Transition
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
	for _, snapshot := range snapshots {
		if snapshot.Root == nil {
			return Data{}, eris.Errorf("alluvial reference %q has no source tree", snapshot.Reference)
		}

		values := selectedValues(snapshot.Root, options)
		data.Columns = append(data.Columns, Column{
			Reference: snapshot.Reference,
			Values:    values,
		})
	}

	data.Transitions = buildTransitions(data.Columns)

	return data, nil
}

func selectedValues(root *model.Directory, options Options) []Value {
	expanded := make(map[string]struct{}, len(options.Expand))
	for _, directory := range options.Expand {
		expanded[path.Clean(directory)] = struct{}{}
	}

	selected := append([]*model.Directory(nil), root.Dirs...)
	for i := 0; i < len(selected); i++ {
		directory := selected[i]
		if directory == nil {
			continue
		}

		if _, ok := expanded[directory.RepoPath]; !ok {
			continue
		}

		selected = append(selected[:i], append(directory.Dirs, selected[i+1:]...)...)
		i--
	}

	values := make([]Value, 0, len(selected))
	for _, directory := range selected {
		if directory == nil || directory.RepoPath == "" {
			continue
		}

		values = append(values, Value{
			Path:  directory.RepoPath,
			Width: directoryWidth(directory, options.Metric),
		})
	}

	slices.SortFunc(values, func(left, right Value) int {
		return cmp.Compare(left.Path, right.Path)
	})

	return values
}

func directoryWidth(directory *model.Directory, metricName metric.Name) float64 {
	if value, ok := directory.Quantity(metricName); ok {
		return float64(value)
	}

	if value, ok := directory.Measure(metricName); ok {
		return value
	}

	return 0
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
