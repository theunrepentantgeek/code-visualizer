package treemap

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// AcquireData runs the data-acquisition stages: scan the filesystem, run
// providers, and populate declarations. Tests that supply a pre-built model
// tree skip this function and inject CommonState.Root directly.
func AcquireData(s *pipeline.State) {
	pipeline.ApplyFuncX(s, stages.ScanFilesystem)
	pipeline.ApplyFuncX(s, stages.CheckGitRequirement)
	pipeline.ApplyFuncX(s, stages.RunProviders)
	pipeline.ApplyFuncX(s, stages.PopulateDeclarations)
}

// RenderPipeline runs every stage from aggregation through writing the canvas.
// It assumes CommonState.Root is populated (by AcquireData in production, or by
// a test harness in golden tests) and that metrics have been resolved.
// Shared by the CLI command and the golden-test harness so both exercise
// identical wiring.
func RenderPipeline(s *pipeline.State) {
	pipeline.ApplyFuncX(s, stages.RunAggregations)
	pipeline.ApplyFuncX(s, stages.FilterBinaryFiles)
	pipeline.ApplyFuncX(s, stages.ExportData)
	pipeline.ApplyFuncX(s, stages.ResolveDimensions)
	pipeline.ApplyFuncX(s, stages.InitDrawingBounds)
	pipeline.ApplyFuncX(s, stages.ReserveTitleBounds)
	pipeline.ApplyFuncX(s, stages.ReserveFooterBounds)
	pipeline.ApplyFuncXY(s, BuildInksStage)
	pipeline.ApplyFuncXYZ(s, BuildLegendStage)
	pipeline.ApplyFuncXY(s, LayoutStage)
	pipeline.ApplyFuncXY(s, RenderStage)
	pipeline.ApplyFuncXYZ(s, LabelStage)
	pipeline.ApplyFuncXY(s, ApplyCanvasBlockLabels)
	pipeline.ApplyFuncX(s, stages.ApplyTitle)
	pipeline.ApplyFuncX(s, stages.ApplyFooter)
	pipeline.ApplyFuncX(s, stages.WriteCanvas)
	pipeline.ApplyFuncXY(s, LogResult)
}
