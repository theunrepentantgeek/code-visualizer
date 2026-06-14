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
		if resolved.SourceLevel != metric.LevelFile {
			return eris.Errorf(
				"aggregation of %s-level metric %q is not yet supported (requires model changes)",
				resolved.SourceLevel, resolved.Expression.Base,
			)
		}

		if err := aggregateDirectory(root, resolved); err != nil {
			return err
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
