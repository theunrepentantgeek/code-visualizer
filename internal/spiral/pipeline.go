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
	s.ApplyFuncXYZ(stages.ScanFilesystem).
		ApplyFuncXY(stages.FilterChangedOnly).
		ApplyFuncX(stages.CheckGitRequirement)
}

func LoadGitMetrics(s *pipeline.State) {
	s.ApplyFuncXYZ(stages.LoadGitMetrics).
		ApplyFuncX(stages.GroupGitHistoryByFile).
		ApplyFuncX(stages.ExtractFileHistory)
}

func LoadFilesystemMetrics(s *pipeline.State) {
	s.ApplyFuncXYZ(stages.RunFilesystemProviders).
		ApplyFuncX(stages.PopulateDeclarations)
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
	s.ApplyFuncX(stages.RunAggregations).
		ApplyFuncX(stages.FilterBinaryFiles).
		ApplyFuncX(stages.PruneFileHistoryToTree).
		ApplyFuncX(stages.ExportData).
		ApplyFuncX(stages.ResolveDimensions).
		ApplyFuncX(stages.InitDrawingBounds).
		ApplyFuncX(stages.ReserveTitleBounds).
		ApplyFuncX(stages.ReserveFooterBounds).
		ApplyFuncXY(BuildTimeBucketsStage).
		ApplyFuncXY(AggregateBucketMetricsStage).
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
