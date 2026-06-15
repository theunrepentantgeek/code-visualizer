package provider

import (
	"strings"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// ResolvedMetric is a fully validated metric ready for computation.
type ResolvedMetric struct {
	Expression       metric.MetricExpression
	Descriptor       BaseMetricDescriptor
	SourceLevel      metric.MetricLevel
	TargetLevel      metric.MetricLevel
	ResultKind       metric.Kind
	ResultName       metric.Name
	NeedsAggregation bool
}

// ResolveExpression validates and resolves a parsed metric expression against
// the global base metric registry for the given target level.
func ResolveExpression(expr metric.MetricExpression, targetLevel metric.MetricLevel) (ResolvedMetric, error) {
	return resolveExpressionWith(globalBaseRegistry, expr, targetLevel)
}

func resolveExpressionWith(
	reg *baseRegistry,
	expr metric.MetricExpression,
	targetLevel metric.MetricLevel,
) (ResolvedMetric, error) {
	desc, ok := reg.get(expr.Base)
	if !ok {
		return ResolvedMetric{}, eris.Errorf("unknown base metric %q", expr.Base)
	}

	if err := validateFilter(desc, expr.Filter); err != nil {
		return ResolvedMetric{}, err
	}

	needsAgg, err := validateAggregation(desc, expr, targetLevel)
	if err != nil {
		return ResolvedMetric{}, err
	}

	resultKind := computeResultKind(desc.Kind, expr.Aggregation)

	return ResolvedMetric{
		Expression:       expr,
		Descriptor:       desc,
		SourceLevel:      desc.Level,
		TargetLevel:      targetLevel,
		ResultKind:       resultKind,
		ResultName:       expr.ResultName(),
		NeedsAggregation: needsAgg,
	}, nil
}

func validateFilter(desc BaseMetricDescriptor, filter metric.FilterName) error {
	if filter.IsZero() {
		return nil
	}

	if !desc.SupportsFilter(filter) {
		if len(desc.Filters) == 0 {
			return eris.Errorf(
				"%q is not a valid filter for %q; %q has no filters",
				filter, desc.Name, desc.Name,
			)
		}

		return eris.Errorf(
			"%q is not a valid filter for %q; valid filters: %s",
			filter, desc.Name, formatFilterNames(desc.Filters),
		)
	}

	return nil
}

func validateAggregation(
	desc BaseMetricDescriptor,
	expr metric.MetricExpression,
	targetLevel metric.MetricLevel,
) (bool, error) {
	needsAgg := !expr.Aggregation.IsZero() || targetLevel != desc.Level

	if !expr.Aggregation.IsZero() && !desc.SupportsAggregation(expr.Aggregation) {
		return false, eris.Errorf(
			"%q is not a valid aggregation for %q; valid aggregations: %s",
			expr.Aggregation, desc.Name, formatAggregationNames(desc.Aggregations),
		)
	}

	if needsAgg && expr.Aggregation.IsZero() {
		return false, eris.Errorf(
			"metric %q requires aggregation at %s level (native level: %s); try: %s",
			desc.Name,
			targetLevel.String(),
			desc.Level.String(),
			formatAggregationSuggestions(desc),
		)
	}

	return needsAgg, nil
}

func computeResultKind(sourceKind metric.Kind, agg metric.AggregationName) metric.Kind {
	switch agg {
	case metric.AggMean:
		return metric.Measure
	case metric.AggCount, metric.AggDistinct:
		return metric.Quantity
	case metric.AggMode:
		return metric.Classification
	default:
		return sourceKind
	}
}

func formatFilterNames(filters []metric.FilterName) string {
	strs := make([]string, len(filters))
	for i, f := range filters {
		strs[i] = string(f)
	}

	return strings.Join(strs, ", ")
}

func formatAggregationNames(aggs []metric.AggregationName) string {
	strs := make([]string, len(aggs))
	for i, a := range aggs {
		strs[i] = string(a)
	}

	return strings.Join(strs, ", ")
}

func formatAggregationSuggestions(desc BaseMetricDescriptor) string {
	return formatAggregationNames(desc.Aggregations)
}
