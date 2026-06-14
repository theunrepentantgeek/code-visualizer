package stages

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// ComputeAggregations walks the directory tree and computes aggregated metric
// values for each resolved expression. Each directory gets its own aggregate
// computed from all descendant source-level nodes.
func ComputeAggregations(root *model.Directory, expressions []provider.ResolvedMetric) error {
	if len(expressions) == 0 {
		return nil
	}

	for _, resolved := range expressions {
		switch resolved.SourceLevel {
		case metric.LevelFile:
			if err := aggregateDirectory(root, resolved); err != nil {
				return err
			}
		case metric.LevelDeclaration:
			if err := aggregateDeclarations(root, resolved); err != nil {
				return err
			}
		case metric.LevelCommit:
			if err := aggregateCommits(root, resolved); err != nil {
				return err
			}
		default:
			return eris.Errorf(
				"aggregation of %s-level metric %q is not supported",
				resolved.SourceLevel, resolved.Expression.Base,
			)
		}
	}

	return nil
}

func aggregateDirectory(dir *model.Directory, resolved provider.ResolvedMetric) error {
	for _, child := range dir.Dirs {
		if err := aggregateDirectory(child, resolved); err != nil {
			return err
		}
	}

	switch resolved.Descriptor.Kind {
	case metric.Classification:
		return aggregateClassification(dir, resolved)
	case metric.Quantity, metric.Measure:
		return aggregateNumeric(dir, resolved)
	default:
		return eris.Errorf(
			"aggregation for metric %q uses unsupported source kind %d",
			resolved.Expression.Base, resolved.Descriptor.Kind,
		)
	}
}

func aggregateNumeric(dir *model.Directory, resolved provider.ResolvedMetric) error {
	values := collectNumericValues(dir, resolved.Expression.Base, resolved.Descriptor.Kind)
	if len(values) == 0 {
		return nil
	}

	result, err := applyNumericAggregation(resolved.Expression.Aggregation, values)
	if err != nil {
		return err
	}

	switch resolved.ResultKind {
	case metric.Quantity:
		dir.SetQuantity(resolved.ResultName, int64(result))
	case metric.Measure:
		dir.SetMeasure(resolved.ResultName, result)
	default:
		return eris.Errorf(
			"aggregation %q for metric %q uses unsupported result kind %d",
			resolved.Expression.Aggregation, resolved.Expression.Base, resolved.ResultKind,
		)
	}

	return nil
}

func aggregateClassification(dir *model.Directory, resolved provider.ResolvedMetric) error {
	values := collectClassificationValues(dir, resolved.Expression.Base)
	if len(values) == 0 {
		return nil
	}

	switch resolved.Expression.Aggregation {
	case metric.AggMode:
		if resolved.ResultKind != metric.Classification {
			return eris.Errorf(
				"aggregation %q for metric %q uses unsupported result kind %d",
				resolved.Expression.Aggregation, resolved.Expression.Base, resolved.ResultKind,
			)
		}

		dir.SetClassification(resolved.ResultName, metric.AggregateMode(values))
	case metric.AggDistinct:
		if resolved.ResultKind != metric.Quantity {
			return eris.Errorf(
				"aggregation %q for metric %q uses unsupported result kind %d",
				resolved.Expression.Aggregation, resolved.Expression.Base, resolved.ResultKind,
			)
		}

		dir.SetQuantity(resolved.ResultName, int64(metric.AggregateDistinct(values)))
	default:
		return eris.Errorf(
			"classification aggregation %q for metric %q is unsupported",
			resolved.Expression.Aggregation, resolved.Expression.Base,
		)
	}

	return nil
}

func collectNumericValues(dir *model.Directory, name metric.Name, kind metric.Kind) []float64 {
	var values []float64

	model.WalkFiles(dir, func(f *model.File) {
		switch kind {
		case metric.Quantity:
			if v, ok := f.Quantity(name); ok {
				values = append(values, float64(v))
			}
		case metric.Measure:
			if v, ok := f.Measure(name); ok {
				values = append(values, v)
			}
		default:
			return
		}
	})

	return values
}

func collectClassificationValues(dir *model.Directory, name metric.Name) []string {
	var values []string

	model.WalkFiles(dir, func(f *model.File) {
		if v, ok := f.Classification(name); ok {
			values = append(values, v)
		}
	})

	return values
}

func applyNumericAggregation(agg metric.AggregationName, values []float64) (float64, error) {
	switch agg {
	case metric.AggSum:
		return metric.AggregateSum(values), nil
	case metric.AggMin:
		return metric.AggregateMin(values), nil
	case metric.AggMax:
		return metric.AggregateMax(values), nil
	case metric.AggMean:
		return metric.AggregateMean(values), nil
	case metric.AggCount:
		return metric.AggregateCount(values), nil
	case metric.AggRange:
		return metric.AggregateRange(values), nil
	default:
		return 0, eris.Errorf("numeric aggregation %q is unsupported", agg)
	}
}

// ---------------------------------------------------------------------------
// Declaration-level aggregation
// ---------------------------------------------------------------------------

func aggregateDeclarations(dir *model.Directory, resolved provider.ResolvedMetric) error {
	for _, child := range dir.Dirs {
		if err := aggregateDeclarations(child, resolved); err != nil {
			return err
		}
	}

	switch resolved.Descriptor.Kind {
	case metric.Classification:
		return aggregateDeclarationClassification(dir, resolved)
	case metric.Quantity, metric.Measure:
		return aggregateDeclarationNumeric(dir, resolved)
	default:
		return eris.Errorf(
			"aggregation for declaration metric %q uses unsupported source kind %d",
			resolved.Expression.Base, resolved.Descriptor.Kind,
		)
	}
}

func aggregateDeclarationNumeric(dir *model.Directory, resolved provider.ResolvedMetric) error {
	values := collectDeclarationNumericValues(dir, resolved)
	if len(values) == 0 {
		return nil
	}

	result, err := applyNumericAggregation(resolved.Expression.Aggregation, values)
	if err != nil {
		return err
	}

	switch resolved.ResultKind {
	case metric.Quantity:
		dir.SetQuantity(resolved.ResultName, int64(result))
	case metric.Measure:
		dir.SetMeasure(resolved.ResultName, result)
	default:
		return eris.Errorf(
			"aggregation %q for declaration metric %q uses unsupported result kind %d",
			resolved.Expression.Aggregation, resolved.Expression.Base, resolved.ResultKind,
		)
	}

	return nil
}

func aggregateDeclarationClassification(dir *model.Directory, resolved provider.ResolvedMetric) error {
	values := collectDeclarationClassificationValues(dir, resolved)
	if len(values) == 0 {
		return nil
	}

	switch resolved.Expression.Aggregation {
	case metric.AggMode:
		dir.SetClassification(resolved.ResultName, metric.AggregateMode(values))
	case metric.AggDistinct:
		dir.SetQuantity(resolved.ResultName, int64(metric.AggregateDistinct(values)))
	default:
		return eris.Errorf(
			"classification aggregation %q for declaration metric %q is unsupported",
			resolved.Expression.Aggregation, resolved.Expression.Base,
		)
	}

	return nil
}

func collectDeclarationNumericValues(dir *model.Directory, resolved provider.ResolvedMetric) []float64 {
	var values []float64

	model.WalkDeclarations(dir, func(d *model.Declaration, _ *model.File) {
		if !resolved.Expression.Filter.IsZero() {
			if !resolved.Descriptor.PassesFilter(resolved.Expression.Filter, d) {
				return
			}
		}

		if !matchesDeclKind(d, resolved.Expression.Base) {
			return
		}

		switch resolved.Descriptor.Kind {
		case metric.Quantity:
			if v, ok := d.Quantity(resolved.Expression.Base); ok {
				values = append(values, float64(v))
			} else if resolved.Expression.Aggregation == metric.AggCount {
				// For count, the declaration itself is the unit being counted
				values = append(values, 1)
			}
		case metric.Measure:
			if v, ok := d.Measure(resolved.Expression.Base); ok {
				values = append(values, v)
			}
		}
	})

	return values
}

func collectDeclarationClassificationValues(dir *model.Directory, resolved provider.ResolvedMetric) []string {
	var values []string

	model.WalkDeclarations(dir, func(d *model.Declaration, _ *model.File) {
		if !resolved.Expression.Filter.IsZero() {
			if !resolved.Descriptor.PassesFilter(resolved.Expression.Filter, d) {
				return
			}
		}

		if !matchesDeclKind(d, resolved.Expression.Base) {
			return
		}

		if v, ok := d.Classification(resolved.Expression.Base); ok {
			values = append(values, v)
		}
	})

	return values
}

// matchesDeclKind checks whether a declaration matches the semantic category
// implied by the base metric name.
func matchesDeclKind(d *model.Declaration, baseName metric.Name) bool {
	switch baseName {
	case "types":
		return d.Kind == "type" || d.Kind == "struct" || d.Kind == "interface"
	case "interfaces":
		return d.Kind == "interface"
	case "structs":
		return d.Kind == "struct"
	case "functions":
		return d.Kind == "function"
	case "methods":
		return d.Kind == "method"
	case "constants":
		return d.Kind == "constant"
	case "variables":
		return d.Kind == "variable"
	case "cyclomatic-complexity", "function-length":
		return d.Kind == "function" || d.Kind == "method"
	default:
		return true
	}
}

// ---------------------------------------------------------------------------
// Commit-level aggregation
// ---------------------------------------------------------------------------

func aggregateCommits(dir *model.Directory, resolved provider.ResolvedMetric) error {
	for _, child := range dir.Dirs {
		if err := aggregateCommits(child, resolved); err != nil {
			return err
		}
	}

	switch resolved.Descriptor.Kind {
	case metric.Quantity, metric.Measure:
		return aggregateCommitNumeric(dir, resolved)
	default:
		return eris.Errorf(
			"aggregation for commit metric %q uses unsupported source kind %d",
			resolved.Expression.Base, resolved.Descriptor.Kind,
		)
	}
}

func aggregateCommitNumeric(dir *model.Directory, resolved provider.ResolvedMetric) error {
	values := collectCommitNumericValues(dir, resolved)
	if len(values) == 0 {
		return nil
	}

	result, err := applyNumericAggregation(resolved.Expression.Aggregation, values)
	if err != nil {
		return err
	}

	switch resolved.ResultKind {
	case metric.Quantity:
		dir.SetQuantity(resolved.ResultName, int64(result))
	case metric.Measure:
		dir.SetMeasure(resolved.ResultName, result)
	default:
		return eris.Errorf(
			"aggregation %q for commit metric %q uses unsupported result kind %d",
			resolved.Expression.Aggregation, resolved.Expression.Base, resolved.ResultKind,
		)
	}

	return nil
}

func collectCommitNumericValues(dir *model.Directory, resolved provider.ResolvedMetric) []float64 {
	var values []float64

	model.WalkCommits(dir, func(c *model.Commit, _ *model.File) {
		switch resolved.Descriptor.Kind {
		case metric.Quantity:
			if v, ok := c.Quantity(resolved.Expression.Base); ok {
				values = append(values, float64(v))
			}
		case metric.Measure:
			if v, ok := c.Measure(resolved.Expression.Base); ok {
				values = append(values, v)
			}
		}
	})

	return values
}
