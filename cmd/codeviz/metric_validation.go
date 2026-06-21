package main

import (
	"strings"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// resolveMetricKind validates that name refers to a known metric and returns
// the kind of the resulting value. The name may be a bare base metric
// (e.g. "file-size") or an aggregation expression (e.g. "declarations.count",
// "public.declarations.count"). Resolution is delegated to provider.ResolveName
// — the single canonical resolver shared with config and the pipeline — at the
// file level, since these metrics are consumed per file. The label describes
// the field being validated (e.g. "size") for error messages.
func resolveMetricKind(label string, name metric.Name) (metric.Kind, error) {
	resolved, err := provider.ResolveName(name, metric.LevelFile)
	if err != nil {
		return 0, friendlyMetricError(label, name, err)
	}

	return resolved.ResultKind, nil
}

// friendlyMetricError turns a provider.ResolveName error into a CLI-friendly
// message. When the base metric is simply unknown, it lists the available
// metrics; otherwise (a bad filter, aggregation, or a metric that needs an
// aggregation) it surfaces the specific resolution failure.
func friendlyMetricError(label string, name metric.Name, err error) error {
	if expr, parseErr := metric.ParseExpression(string(name)); parseErr == nil {
		if _, ok := provider.GetBase(expr.Base); ok {
			return eris.Wrapf(err, "invalid %s metric %q", label, name)
		}
	}

	return eris.Errorf(
		"unknown %s metric %q; available metrics: %s", label, name, formatMetricNames(),
	)
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

// formatMetricNames returns a comma-separated list of all registered base
// metric names, used in "available metrics" error messages.
func formatMetricNames() string {
	names := provider.BaseNames()
	strs := make([]string, len(names))

	for i, n := range names {
		strs[i] = string(n)
	}

	return strings.Join(strs, ", ")
}
