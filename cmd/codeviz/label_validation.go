package main

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

type selectedMetricRole struct {
	name   string
	metric metric.Name
}

type labelMetricResolver func(metric.Name) (metric.Name, error)

func validateLabelMetrics(
	labels []metric.Name,
	resolve labelMetricResolver,
	roles ...selectedMetricRole,
) error {
	selected := make(map[metric.Name]selectedMetricRole, len(roles))
	for _, role := range roles {
		if role.metric != "" {
			selected[role.metric] = role
		}
	}

	seen := make(map[metric.Name]struct{}, len(labels))
	for _, label := range labels {
		resolved, err := resolve(label)
		if err != nil {
			return friendlyMetricError("label", label, err)
		}

		if _, exists := seen[resolved]; exists {
			return eris.Errorf("label metric %q is repeated", label)
		}

		seen[resolved] = struct{}{}

		if role, exists := selected[resolved]; exists {
			return eris.Errorf(
				"label metric %q duplicates selected %s role metric %q",
				label,
				role.name,
				role.metric,
			)
		}
	}

	return nil
}

func resolveFileLabelMetric(name metric.Name) (metric.Name, error) {
	resolved, err := provider.ResolveForValidation(name)
	if err != nil {
		return "", eris.Wrap(err, "resolve label metric")
	}

	return resolved.ResultName, nil
}

// resolveDirectoryLabelMetric resolves a label metric for directory-level
// visualizations. Temporal modifiers are rejected because label acquisition and
// rendering only support single-snapshot values.
func resolveDirectoryLabelMetric(name metric.Name) (metric.Name, error) {
	expression, err := metric.ParseExpression(string(name))
	if err != nil {
		return "", eris.Wrap(err, "parse label metric")
	}

	if !expression.Temporal.IsZero() {
		return "", eris.Errorf(
			"temporal modifier %q is not supported for label metrics",
			expression.Temporal,
		)
	}

	return resolveDirectoryExpression(expression)
}

// resolveDirectoryRoleMetric resolves a selected role metric (size, fill, border,
// width) to the per-snapshot metric it is computed from, so temporal roles still
// collide with labels naming that underlying metric.
func resolveDirectoryRoleMetric(name metric.Name) (metric.Name, error) {
	expression, err := metric.ParseExpression(string(name))
	if err != nil {
		return "", eris.Wrap(err, "parse role metric")
	}

	return resolveDirectoryExpression(expression.WithoutTemporal())
}

func resolveDirectoryExpression(expression metric.MetricExpression) (metric.Name, error) {
	descriptor, ok := provider.GetBase(expression.Base)
	if !ok {
		return "", eris.Errorf("unknown base metric %q", expression.Base)
	}

	if expression.Aggregation.IsZero() {
		aggregation, ok := descriptor.Kind.DefaultAggregation()
		if !ok {
			return "", eris.Errorf("unsupported metric kind %d", descriptor.Kind)
		}

		expression.Aggregation = aggregation
	}

	resolved, err := provider.ResolveExpression(expression, metric.LevelDirectory)
	if err != nil {
		return "", eris.Wrap(err, "resolve label metric")
	}

	return resolved.ResultName, nil
}

func resolvedRole(
	name string,
	value metric.Name,
	resolve labelMetricResolver,
) selectedMetricRole {
	if value == "" {
		return selectedMetricRole{name: name}
	}

	resolved, err := resolve(value)
	if err != nil {
		return selectedMetricRole{name: name}
	}

	return selectedMetricRole{name: name, metric: resolved}
}
