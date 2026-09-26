package donuttree

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// AcquireData runs scan, providers, and declaration population.
func AcquireData(s *pipeline.State) {
	pipeline.ApplyFuncXYZ(s, stages.ScanFilesystem)
	pipeline.ApplyFuncXY(s, stages.FilterChangedOnly)
	pipeline.ApplyFuncX(s, stages.CheckGitRequirement)
	pipeline.ApplyFuncXYZ(s, stages.PrewarmGitMetrics)
	pipeline.ApplyFuncXYZ(s, stages.RunProviders)
	pipeline.ApplyFuncX(s, stages.PopulateDeclarations)
}

// RenderPipeline runs the donut tree pipeline after metrics and data are available.
func RenderPipeline(s *pipeline.State) {
	RenderVisualization(s)
	WriteOutput(s)
}

func RenderVisualization(s *pipeline.State) {
	pipeline.ApplyFuncX(s, stages.RunAggregations)
	pipeline.ApplyFuncX(s, stages.FilterBinaryFiles)
	pipeline.ApplyFuncX(s, stages.ExportData)
	pipeline.ApplyFuncX(s, stages.ResolveDimensions)
	pipeline.ApplyFuncX(s, stages.InitDrawingBounds)
	pipeline.ApplyFuncX(s, stages.ReserveTitleBounds)
	pipeline.ApplyFuncX(s, stages.ReserveFooterBounds)
	pipeline.ApplyFuncXY(s, BuildInksStage)
	pipeline.ApplyFuncXY(s, BuildLegendStage)
	pipeline.ApplyFuncXY(s, LayoutStage)
	pipeline.ApplyFuncXY(s, RenderStage)
	pipeline.ApplyFuncX(s, stages.ApplyTitle)
	pipeline.ApplyFuncX(s, stages.ApplyFooter)
}

func WriteOutput(s *pipeline.State) {
	pipeline.ApplyFuncX(s, stages.WriteCanvas)
}
