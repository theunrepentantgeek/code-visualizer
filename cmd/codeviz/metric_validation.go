package main

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// resolveMetricKind validates that name refers to a known metric and returns
// the kind of the resulting value. The name may be a bare base metric
// (e.g. "file-size") or an aggregation expression (e.g. "declarations.count",
// "public.declarations.count"); expressions are resolved against the metric
// registry so aggregated metrics are accepted wherever a metric is expected.
// The label describes the field being validated (e.g. "size") for error messages.
func resolveMetricKind(label string, name metric.Name) (metric.Kind, error) {
	expr, parseErr := metric.ParseExpression(string(name))

	// Names carrying a filter or aggregation are expressions; resolve them to
	// obtain the kind of the computed value.
	if parseErr == nil && (!expr.Filter.IsZero() || !expr.Aggregation.IsZero()) {
		resolved, resolveErr := provider.ResolveExpression(expr, metric.LevelFile)
		if resolveErr != nil {
			return 0, eris.Wrapf(resolveErr, "invalid %s metric %q", label, name)
		}

		return resolved.ResultKind, nil
	}

	d, ok := provider.GetBase(name)
	if !ok {
		return 0, eris.Errorf(
			"unknown %s metric %q; available metrics: %s", label, name, formatMetricNames(),
		)
	}

	return d.Kind, nil
}

// validateNumericMetric validates that name refers to a numeric metric, or an
// aggregation expression producing a numeric value (quantity or measure).
func validateNumericMetric(label string, name metric.Name) error {
	kind, err := resolveMetricKind(label, name)
	if err != nil {
		return err
	}

	if kind != metric.Quantity && kind != metric.Measure {
		return eris.Errorf("%s metric must be numeric, got %q (kind: %d)", label, name, kind)
	}

	return nil
}

// validateMetricExists validates that name refers to a known metric of any
// kind, accepting both base metrics and aggregation expressions. Used for
// fields (such as scatter axes) that accept classification and numeric metrics.
func validateMetricExists(label string, name metric.Name) error {
	_, err := resolveMetricKind(label, name)

	return err
}
