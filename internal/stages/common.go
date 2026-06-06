// Package stages provides shared visualization-pipeline stages and the
// CommonState type that they operate on. Viz-specific state is kept
// alongside CommonState in the type-keyed *pipeline.State.
package stages

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
)

// Flags is the cross-cutting flag bundle passed to every viz command's Run.
// It mirrors cmd/codeviz.Flags but lives here so this package does not
// depend on package main. The orchestrator constructs one and assigns it
// into CommonState.Flags before running the pipeline.
type Flags struct {
	Quiet        bool
	Verbose      bool
	Debug        bool
	ExportConfig string
	ExportData   string
	Config       *config.Config
}

// CommonState contains fields used by shared stages. Each viz pipeline
// stores a *CommonState alongside the per-viz state and config in the
// type-keyed *pipeline.State.
type CommonState struct {
	// Inputs: set by the orchestrator before applying any stages.
	TargetPath string
	Output     string
	Flags      *Flags
	RootConfig *config.Config
	VizName    string // active visualization name for export trimming
	CLIFilters []filter.Rule

	// Populated by shared stages during the pipeline:
	FilterRules []filter.Rule    // BuildFilterRules
	Requested   []metric.Name    // viz-specific ResolveMetrics
	Root        *model.Directory // ScanFilesystem
	Width       int              // ResolveDimensions
	Height      int              // ResolveDimensions
	Canvas      *canvas.Canvas   // viz-specific Render

	// Git history (populated by LoadGitHistory / GroupGitHistoryByFile / ExtractFileHistory).
	// GitHistory is written once and not mutated afterward; consumers may hold
	// *Commit references for the lifetime of CommonState.
	GitHistory    []git.Commit
	FileHistory   map[*model.File][]CommitRef
	FileTimeRange map[*model.File]TimeRange
}
