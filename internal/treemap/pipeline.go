package treemap

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// AcquireData runs the data-acquisition stages: scan the filesystem, run
// providers, and populate declarations. Tests that supply a pre-built model
// tree skip this function and inject CommonState.Root directly.
func AcquireData(s *pipeline.State) {
	ScanData(s)
	LoadGitMetrics(s)
	LoadFilesystemMetrics(s)
}

func ScanData(s *pipeline.State) {
	s.ApplyFuncXYZ(stages.ScanFilesystem)
	s.ApplyFuncXY(stages.FilterChangedOnly)
	s.ApplyFuncX(stages.CheckGitRequirement)
}

func LoadGitMetrics(s *pipeline.State) {
	s.ApplyFuncXYZ(stages.LoadGitMetrics)
}

func LoadFilesystemMetrics(s *pipeline.State) {
	s.ApplyFuncXYZ(stages.RunFilesystemProviders)
	s.ApplyFuncX(stages.PopulateDeclarations)
}

// RenderPipeline runs every stage from aggregation through writing the canvas.
// It assumes CommonState.Root is populated (by AcquireData in production, or by
// a test harness in golden tests) and that metrics have been resolved.
// Shared by the CLI command and the golden-test harness so both exercise
// identical wiring.
func RenderPipeline(s *pipeline.State) {
	RenderVisualization(s)
	WriteOutput(s)
}

func RenderVisualization(s *pipeline.State) {
	s.ApplyFuncX(stages.RunAggregations)
	s.ApplyFuncX(stages.FilterBinaryFiles)
	s.ApplyFuncX(stages.ExportData)
	s.ApplyFuncX(stages.ResolveDimensions)
	s.ApplyFuncX(stages.InitDrawingBounds)
	s.ApplyFuncX(stages.ReserveTitleBounds)
	s.ApplyFuncX(stages.ReserveFooterBounds)
	s.ApplyFuncXY(BuildInksStage)
	s.ApplyFuncXYZ(BuildLegendStage)
	s.ApplyFuncXY(LayoutStage)
	s.ApplyFuncXY(RenderStage)
	s.ApplyFuncXYZ(LabelStage)
	s.ApplyFuncXY(ApplyCanvasBlockLabels)
	s.ApplyFuncX(stages.ApplyTitle)
	s.ApplyFuncX(stages.ApplyFooter)
}

func WriteOutput(s *pipeline.State) {
	s.ApplyFuncX(stages.WriteCanvas)
}
