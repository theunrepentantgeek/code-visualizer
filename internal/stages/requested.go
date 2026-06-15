package stages

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// RequestedMetrics separates user-requested metric names into expressions
// that need aggregation and legacy names that go directly to provider.Run.
type RequestedMetrics struct {
	// BaseMetrics are the base metric names extracted from resolved expressions.
	// These must be run by provider.Run to populate source-level data.
	BaseMetrics []metric.Name
	// Expressions are fully resolved metrics that need aggregation computation.
	Expressions []provider.ResolvedMetric
	// Legacy are metric names that couldn't be parsed/resolved as expressions
	// and should be passed to provider.Run as-is (backward compatibility).
	Legacy []metric.Name
}

// LegacyNames returns all metric names that should be passed to provider.Run:
// both the base metrics needed by expressions AND the legacy unresolved names.
func (r RequestedMetrics) LegacyNames() []metric.Name {
	seen := make(map[metric.Name]bool, len(r.BaseMetrics)+len(r.Legacy))

	var result []metric.Name

	for _, n := range r.BaseMetrics {
		if !seen[n] {
			seen[n] = true
			result = append(result, n)
		}
	}

	for _, n := range r.Legacy {
		if !seen[n] {
			seen[n] = true
			result = append(result, n)
		}
	}

	return result
}

// HasDeclarationExpressions reports whether any expression needs declaration-level data.
func (r RequestedMetrics) HasDeclarationExpressions() bool {
	for _, expr := range r.Expressions {
		if expr.SourceLevel == metric.LevelDeclaration {
			return true
		}
	}

	return false
}

// HasCommitExpressions reports whether any expression needs commit-level data.
func (r RequestedMetrics) HasCommitExpressions() bool {
	for _, expr := range r.Expressions {
		if expr.SourceLevel == metric.LevelCommit {
			return true
		}
	}

	return false
}

// DescriptorFor returns a MetricDescriptor for the given metric name by
// checking resolved expressions first, then falling back to the legacy
// provider registry. This allows the Ink/rendering layer to understand
// expression-computed metrics (e.g. "public.methods.count") that don't
// exist in the legacy registry.
func (r RequestedMetrics) DescriptorFor(name metric.Name) (provider.MetricDescriptor, bool) {
	for i := range r.Expressions {
		if r.Expressions[i].ResultName == name {
			return provider.MetricDescriptor{
				Name: name,
				Kind: r.Expressions[i].ResultKind,
			}, true
		}
	}

	return provider.GetDescriptor(name, metric.File)
}

// ClassifyRequestedMetrics takes a flat list of metric name strings and
// classifies each as either a resolvable expression or a legacy metric name.
func ClassifyRequestedMetrics(names []metric.Name, targetLevel metric.MetricLevel) RequestedMetrics {
	var result RequestedMetrics

	baseSeen := make(map[metric.Name]bool)

	for _, name := range names {
		expr, parseErr := metric.ParseExpression(string(name))
		if parseErr != nil {
			result.Legacy = append(result.Legacy, name)

			continue
		}

		resolved, resolveErr := provider.ResolveExpression(expr, targetLevel)
		if resolveErr != nil {
			result.Legacy = append(result.Legacy, name)

			continue
		}

		if !resolved.NeedsAggregation {
			result.Legacy = append(result.Legacy, name)

			continue
		}

		result.Expressions = append(result.Expressions, resolved)

		// Only add to BaseMetrics if the source is file-level (needs provider.Run).
		// Declaration and commit level metrics are populated by separate stages.
		if resolved.SourceLevel == metric.LevelFile && !baseSeen[expr.Base] {
			baseSeen[expr.Base] = true
			result.BaseMetrics = append(result.BaseMetrics, expr.Base)
		}
	}

	return result
}
