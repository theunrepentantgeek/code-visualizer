package stages

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// RequestedMetrics separates user-requested metric names into base metrics
// that load directly from providers and expressions that need aggregation.
type RequestedMetrics struct {
	// BaseMetrics are the base metric names that must be loaded from providers.
	BaseMetrics []metric.Name
	// Expressions are fully resolved metrics that need aggregation computation.
	Expressions []provider.ResolvedMetric
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
// checking resolved expressions first, then falling back to the provider
// registry. This allows the Ink/rendering layer to understand
// expression-computed metrics (e.g. "public.methods.count") that don't
// exist in the provider registry.
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

func appendBaseMetric(result *RequestedMetrics, baseSeen map[metric.Name]bool, name metric.Name) {
	if baseSeen[name] {
		return
	}

	baseSeen[name] = true
	result.BaseMetrics = append(result.BaseMetrics, name)
}

// ClassifyRequestedMetrics takes a flat list of metric names and classifies
// each as either a resolvable expression or a base metric name.
func ClassifyRequestedMetrics(names []metric.Name, targetLevel metric.MetricLevel) RequestedMetrics {
	var result RequestedMetrics

	baseSeen := make(map[metric.Name]bool)

	for _, name := range names {
		expr, parseErr := metric.ParseExpression(string(name))
		if parseErr != nil {
			appendBaseMetric(&result, baseSeen, name)

			continue
		}

		resolved, resolveErr := provider.ResolveExpression(expr, targetLevel)
		if resolveErr != nil {
			appendBaseMetric(&result, baseSeen, name)

			continue
		}

		if !resolved.NeedsAggregation {
			appendBaseMetric(&result, baseSeen, expr.Base)

			continue
		}

		result.Expressions = append(result.Expressions, resolved)

		// Only add to BaseMetrics if the source is file-level (needs RunLoaders).
		// Declaration and commit level metrics are populated by separate stages.
		if resolved.SourceLevel == metric.LevelFile {
			appendBaseMetric(&result, baseSeen, expr.Base)
		}
	}

	return result
}
