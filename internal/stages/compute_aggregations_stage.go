package stages

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/golang"
)

// PopulateDeclarations walks all Go files and populates per-declaration model
// nodes. Must run after RunProviders and before RunAggregations.
func PopulateDeclarations(c *CommonState) error {
	if !c.Requested.HasDeclarationExpressions() {
		return nil
	}

	model.WalkFiles(c.Root, func(f *model.File) {
		golang.PopulateDeclarations(f)
	})

	return nil
}

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
