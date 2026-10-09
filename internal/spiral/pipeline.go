package spiral

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// AcquireData runs scan, providers, declaration population, and git-history
// loading. The git-history stages populate CommonState.FileHistory and
// FileTimeRange, which the render pipeline's time-bucket stages consume. Tests
// that supply synthetic history set those fields directly and skip AcquireData.
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
	s.ApplyFuncX(stages.GroupGitHistoryByFile)
	s.ApplyFuncX(stages.ExtractFileHistory)
}

func LoadFilesystemMetrics(s *pipeline.State) {
	s.ApplyFuncXYZ(stages.RunFilesystemProviders)
	s.ApplyFuncX(stages.PopulateDeclarations)
}

// RenderPipeline runs aggregation through writing the canvas, assuming
// CommonState.Root, the resolved metrics, CommonState.FileHistory and
// CommonState.FileTimeRange are populated. Shared by the CLI command and the
// golden-test harness so both exercise identical wiring.
func RenderPipeline(s *pipeline.State) {
	RenderVisualization(s)
	WriteOutput(s)
}

func RenderVisualization(s *pipeline.State) {
	s.ApplyFuncX(stages.RunAggregations)
	s.ApplyFuncX(stages.FilterBinaryFiles)
	s.ApplyFuncX(stages.PruneFileHistoryToTree)
	s.ApplyFuncX(stages.ExportData)
	s.ApplyFuncX(stages.ResolveDimensions)
	s.ApplyFuncX(stages.InitDrawingBounds)
	s.ApplyFuncX(stages.ReserveTitleBounds)
	s.ApplyFuncX(stages.ReserveFooterBounds)
	s.ApplyFuncXY(BuildTimeBucketsStage)
	s.ApplyFuncXY(AggregateBucketMetricsStage)
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
