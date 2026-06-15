package provider

import (
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// ProviderDescriptor declares the shared metadata for a metric provider package,
// including the filter vocabulary available to all metrics in that provider.
type ProviderDescriptor struct {
	Name    string
	Filters map[metric.FilterName]string
}

// HasFilter reports whether this provider defines the given filter name.
func (pd ProviderDescriptor) HasFilter(name metric.FilterName) bool {
	_, ok := pd.Filters[name]

	return ok
}

// BaseMetricDescriptor is the static metadata for a composable base metric.
type BaseMetricDescriptor struct {
	Name           metric.Name
	Kind           metric.Kind
	Level          metric.MetricLevel
	Description    string
	Filters        []metric.FilterName
	Aggregations   []metric.AggregationName
	Dependencies   []metric.Name
	DefaultPalette palette.PaletteName
	FilterFunc     func(filter metric.FilterName, node any) bool
}

// PassesFilter evaluates the FilterFunc for the given filter and node.
// Returns true if no FilterFunc is registered (no filtering applied).
func (d BaseMetricDescriptor) PassesFilter(filter metric.FilterName, node any) bool {
	if d.FilterFunc == nil {
		return true
	}

	return d.FilterFunc(filter, node)
}

// SupportsAggregation reports whether this base metric declares the given
// aggregation as valid.
func (d BaseMetricDescriptor) SupportsAggregation(agg metric.AggregationName) bool {
	return slices.Contains(d.Aggregations, agg)
}

// SupportsFilter reports whether this base metric declares the given filter
// as valid.
func (d BaseMetricDescriptor) SupportsFilter(filter metric.FilterName) bool {
	return slices.Contains(d.Filters, filter)
}
