package metric

// AggregationName identifies an aggregation function (e.g., "sum", "max", "mean").
type AggregationName string

// Standard aggregation names.
const (
	AggSum      AggregationName = "sum"
	AggMin      AggregationName = "min"
	AggMax      AggregationName = "max"
	AggMean     AggregationName = "mean"
	AggCount    AggregationName = "count"
	AggMode     AggregationName = "mode"
	AggDistinct AggregationName = "distinct"
	AggRange    AggregationName = "range"
)

// knownAggregations is the fixed set of valid aggregation verbs.
var knownAggregations = map[AggregationName]struct{}{
	AggSum:      {},
	AggMin:      {},
	AggMax:      {},
	AggMean:     {},
	AggCount:    {},
	AggMode:     {},
	AggDistinct: {},
	AggRange:    {},
}

// IsZero reports whether the aggregation name is empty (no aggregation).
func (a AggregationName) IsZero() bool {
	return a == ""
}

// IsKnown reports whether the aggregation name is one of the recognized verbs.
func (a AggregationName) IsKnown() bool {
	_, ok := knownAggregations[a]

	return ok
}
