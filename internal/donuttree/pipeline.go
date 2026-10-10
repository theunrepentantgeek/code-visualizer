package donuttree

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
	s.ApplyFuncXYZ(stages.ScanFilesystem).
		ApplyFuncXY(stages.FilterChangedOnly).
		ApplyFuncX(stages.CheckGitRequirement)
}

func LoadGitMetrics(s *pipeline.State) {
	s.ApplyFuncXYZ(stages.LoadGitMetrics)
}

func LoadFilesystemMetrics(s *pipeline.State) {
	s.ApplyFuncXYZ(stages.RunFilesystemProviders).
		ApplyFuncX(stages.PopulateDeclarations)
}

// RenderPipeline runs the donut tree pipeline after metrics and data are available.
func RenderPipeline(s *pipeline.State) {
	RenderVisualization(s)
	WriteOutput(s)
}

func RenderVisualization(s *pipeline.State) {
	s.ApplyFuncX(stages.RunAggregations).
		ApplyFuncX(stages.FilterBinaryFiles).
		ApplyFuncX(stages.ExportData).
		ApplyFuncX(stages.ResolveDimensions).
		ApplyFuncX(stages.InitDrawingBounds).
		ApplyFuncX(stages.ReserveTitleBounds).
		ApplyFuncX(stages.ReserveFooterBounds).
		ApplyFuncXY(BuildInksStage).
		ApplyFuncXY(BuildLegendStage).
		ApplyFuncXY(LayoutStage).
		ApplyFuncXY(RenderStage).
		ApplyFuncX(stages.ApplyTitle).
		ApplyFuncX(stages.ApplyFooter)
}

func WriteOutput(s *pipeline.State) {
	s.ApplyFuncX(stages.WriteCanvas)
}
