package stages

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// fileLevelLookupKey returns the storage key for a file-level metric value.
// For filtered expressions (e.g. "stdlib.imports.sum"), the value is stored
// under the filter.base form ("stdlib.imports"), not the bare base name.
func fileLevelLookupKey(expr metric.MetricExpression) metric.Name {
	if expr.Filter.IsZero() {
		return expr.Base
	}

	return metric.MetricExpression{Filter: expr.Filter, Base: expr.Base}.ResultName()
}

// ComputeAggregations walks the directory tree and computes aggregated metric
// values for each resolved expression. Each directory gets its own aggregate
// computed from all descendant source-level nodes.
func ComputeAggregations(root *model.Directory, expressions []provider.ResolvedMetric) error {
	for _, resolved := range expressions {
		if err := computeOneAggregation(root, resolved); err != nil {
			return err
		}
	}

	return nil
}

func computeOneAggregation(root *model.Directory, resolved provider.ResolvedMetric) error {
	switch resolved.SourceLevel {
	case metric.LevelFile:
		return aggregateDirectory(root, resolved)
	case metric.LevelDeclaration:
		return aggregateDeclarations(root, resolved)
	case metric.LevelCommit:
		return aggregateCommits(root, resolved)
	default:
		return eris.Errorf(
			"aggregation of %s-level metric %q is not supported",
			resolved.SourceLevel, resolved.Expression.Base,
		)
	}
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
	lookupKey := fileLevelLookupKey(resolved.Expression)
	values := collectNumericValues(dir, lookupKey, resolved.Descriptor.Kind)

	if len(values) == 0 {
		return nil
	}

	return applyAndStoreNumeric(dir, resolved, values)
}

func aggregateClassification(dir *model.Directory, resolved provider.ResolvedMetric) error {
	lookupKey := fileLevelLookupKey(resolved.Expression)
	values := collectClassificationValues(dir, lookupKey)

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

// metricStorer is the subset of MetricContainer needed by applyAndStoreNumeric.
type metricStorer interface {
	SetQuantity(name metric.Name, v int64)
	SetMeasure(name metric.Name, v float64)
}

// applyAndStoreNumeric computes the aggregation and stores the result on the container.
func applyAndStoreNumeric(container metricStorer, resolved provider.ResolvedMetric, values []float64) error {
	result, err := applyNumericAggregation(resolved.Expression.Aggregation, values)
	if err != nil {
		return err
	}

	switch resolved.ResultKind {
	case metric.Quantity:
		container.SetQuantity(resolved.ResultName, int64(result))
	case metric.Measure:
		container.SetMeasure(resolved.ResultName, result)
	default:
		return eris.Errorf(
			"aggregation %q for metric %q uses unsupported result kind %d",
			resolved.Expression.Aggregation, resolved.Expression.Base, resolved.ResultKind,
		)
	}

	return nil
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

// aggregateDeclarations computes per-file aggregations from each file's
// declarations (so per-file consumers like bubbletree borders can read them),
// then computes directory values from all descendant declarations directly.
func aggregateDeclarations(dir *model.Directory, resolved provider.ResolvedMetric) error {
	for _, child := range dir.Dirs {
		if err := aggregateDeclarations(child, resolved); err != nil {
			return err
		}
	}

	// Step 1: aggregate declarations within each file individually.
	for _, f := range dir.Files {
		aggregateFileDeclarations(f, resolved)
	}

	// Step 2: aggregate all descendant declarations (flat) for the directory.
	return aggregateDirectoryDeclarations(dir, resolved)
}

// aggregateFileDeclarations computes the aggregation across a single file's
// declarations and stores the result on that file.
func aggregateFileDeclarations(f *model.File, resolved provider.ResolvedMetric) {
	switch resolved.Descriptor.Kind {
	case metric.Classification:
		aggregateFileDeclarationClassification(f, resolved)
	default:
		aggregateFileDeclarationNumeric(f, resolved)
	}
}

func aggregateFileDeclarationNumeric(f *model.File, resolved provider.ResolvedMetric) {
	values := collectFileDeclarationNumericValues(f, resolved)
	if len(values) == 0 {
		return
	}

	// Silently skip errors at file level since aggregateDirectoryDeclarationNumeric
	// will report them for the directory.
	_ = applyAndStoreNumeric(f, resolved, values)
}

func aggregateFileDeclarationClassification(f *model.File, resolved provider.ResolvedMetric) {
	values := collectFileDeclarationClassificationValues(f, resolved)
	if len(values) == 0 {
		return
	}

	switch resolved.Expression.Aggregation {
	case metric.AggMode:
		f.SetClassification(resolved.ResultName, metric.AggregateMode(values))
	case metric.AggDistinct:
		f.SetQuantity(resolved.ResultName, int64(metric.AggregateDistinct(values)))
	default:
		// unsupported classification aggregation — skip
	}
}

// aggregateDirectoryDeclarations computes the directory-level value from all
// descendant declarations directly (flat aggregation preserving correct
// semantics for mean/count/etc).
func aggregateDirectoryDeclarations(dir *model.Directory, resolved provider.ResolvedMetric) error {
	switch resolved.Descriptor.Kind {
	case metric.Classification:
		return aggregateDirectoryDeclarationClassification(dir, resolved)
	default:
		return aggregateDirectoryDeclarationNumeric(dir, resolved)
	}
}

func aggregateDirectoryDeclarationClassification(dir *model.Directory, resolved provider.ResolvedMetric) error {
	values := collectAllDeclarationClassificationValues(dir, resolved)
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

func aggregateDirectoryDeclarationNumeric(dir *model.Directory, resolved provider.ResolvedMetric) error {
	values := collectAllDeclarationNumericValues(dir, resolved)
	if len(values) == 0 {
		return nil
	}

	return applyAndStoreNumeric(dir, resolved, values)
}

func collectFileDeclarationNumericValues(f *model.File, resolved provider.ResolvedMetric) []float64 {
	var values []float64

	for _, d := range f.Declarations {
		if !declarationMatchesExpression(d, resolved) {
			continue
		}

		if v, ok := declarationNumericValue(d, resolved); ok {
			values = append(values, v)
		}
	}

	return values
}

func collectFileDeclarationClassificationValues(f *model.File, resolved provider.ResolvedMetric) []string {
	var values []string

	for _, d := range f.Declarations {
		if !declarationMatchesExpression(d, resolved) {
			continue
		}

		if v, ok := d.Classification(resolved.Expression.Base); ok {
			values = append(values, v)
		}
	}

	return values
}

func collectAllDeclarationNumericValues(dir *model.Directory, resolved provider.ResolvedMetric) []float64 {
	var values []float64

	model.WalkDeclarations(dir, func(d *model.Declaration, _ *model.File) {
		if !declarationMatchesExpression(d, resolved) {
			return
		}

		if v, ok := declarationNumericValue(d, resolved); ok {
			values = append(values, v)
		}
	})

	return values
}

func collectAllDeclarationClassificationValues(dir *model.Directory, resolved provider.ResolvedMetric) []string {
	var values []string

	model.WalkDeclarations(dir, func(d *model.Declaration, _ *model.File) {
		if !declarationMatchesExpression(d, resolved) {
			return
		}

		if v, ok := d.Classification(resolved.Expression.Base); ok {
			values = append(values, v)
		}
	})

	return values
}

func declarationNumericValue(d *model.Declaration, resolved provider.ResolvedMetric) (float64, bool) {
	switch resolved.Descriptor.Kind {
	case metric.Quantity:
		if v, ok := d.Quantity(resolved.Expression.Base); ok {
			return float64(v), true
		}

		if resolved.Expression.Aggregation == metric.AggCount {
			return 1, true
		}
	case metric.Measure:
		if v, ok := d.Measure(resolved.Expression.Base); ok {
			return v, true
		}
	default:
		// Classification metrics use classification collectors
	}

	return 0, false
}

// declarationMatchesExpression returns true if the declaration passes the
// filter and kind checks required by the resolved expression.
func declarationMatchesExpression(d *model.Declaration, resolved provider.ResolvedMetric) bool {
	if !resolved.Expression.Filter.IsZero() {
		if !resolved.Descriptor.PassesFilter(resolved.Expression.Filter, d) {
			return false
		}
	}

	return resolved.Descriptor.MatchesDeclKind(d.Kind)
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

	return applyAndStoreNumeric(dir, resolved, values)
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
		default:
			// Commit-level classification metrics not yet supported
		}
	})

	return values
}
