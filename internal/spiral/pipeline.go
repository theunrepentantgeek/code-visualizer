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
	pipeline.ApplyFuncX(s, stages.ScanFilesystem)
	pipeline.ApplyFuncX(s, stages.CheckGitRequirement)
	pipeline.ApplyFuncX(s, stages.RunProviders)
	pipeline.ApplyFuncX(s, stages.PopulateDeclarations)
	pipeline.ApplyFuncX(s, stages.LoadGitHistory)
	pipeline.ApplyFuncX(s, stages.GroupGitHistoryByFile)
	pipeline.ApplyFuncX(s, stages.ExtractFileHistory)
}

// RenderPipeline runs aggregation through writing the canvas, assuming
// CommonState.Root, the resolved metrics, CommonState.FileHistory and
// CommonState.FileTimeRange are populated. Shared by the CLI command and the
// golden-test harness so both exercise identical wiring.
func RenderPipeline(s *pipeline.State) {
	pipeline.ApplyFuncX(s, stages.RunAggregations)
	pipeline.ApplyFuncX(s, stages.FilterBinaryFiles)
	pipeline.ApplyFuncX(s, stages.PruneFileHistoryToTree)
	pipeline.ApplyFuncX(s, stages.ExportData)
	pipeline.ApplyFuncX(s, stages.ResolveDimensions)
	pipeline.ApplyFuncX(s, stages.InitDrawingBounds)
	pipeline.ApplyFuncX(s, stages.ReserveTitleBounds)
	pipeline.ApplyFuncX(s, stages.ReserveFooterBounds)
	pipeline.ApplyFuncXY(s, BuildTimeBucketsStage)
	pipeline.ApplyFuncXY(s, AggregateBucketMetricsStage)
	pipeline.ApplyFuncXY(s, BuildInksStage)
	pipeline.ApplyFuncXY(s, BuildLegendStage)
	pipeline.ApplyFuncXY(s, LayoutStage)
	pipeline.ApplyFuncXY(s, RenderStage)
	pipeline.ApplyFuncX(s, stages.ApplyTitle)
	pipeline.ApplyFuncX(s, stages.ApplyFooter)
	pipeline.ApplyFuncX(s, stages.WriteCanvas)
	pipeline.ApplyFuncXY(s, LogResult)
}
