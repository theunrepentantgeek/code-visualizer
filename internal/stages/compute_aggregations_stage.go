package stages

import "github.com/rotisserie/eris"

// RunAggregations is the pipeline stage that computes aggregated metrics
// for all expressions in c.Requested.Expressions.
func RunAggregations(c *CommonState) error {
	if len(c.Requested.Expressions) == 0 {
		return nil
	}

	return eris.Wrap(
		ComputeAggregations(c.Root, c.Requested.Expressions),
		"failed to compute metric aggregations",
	)
}
