package scatter

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// AcquireData runs scan, providers, and declaration population.
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

// RenderPipeline runs aggregation through writing the canvas, assuming
// CommonState.Root and the resolved metrics are populated. Shared by the CLI
// command and the golden-test harness so both exercise identical wiring.
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
	s.ApplyFuncXY(BuildLegendStage)
	s.ApplyFuncXY(LayoutStage)
	s.ApplyFuncXY(RenderStage)
	s.ApplyFuncX(stages.ApplyTitle)
	s.ApplyFuncX(stages.ApplyFooter)
}

func WriteOutput(s *pipeline.State) {
	s.ApplyFuncX(stages.WriteCanvas)
}
