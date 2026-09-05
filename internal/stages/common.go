// Package stages provides shared visualization-pipeline stages and the
// CommonState type that they operate on. Viz-specific state is kept
// alongside CommonState in the type-keyed *pipeline.State.
package stages

import (
	"time"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
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
	From         time.Time
	Until        time.Time
}

// CommonState contains fields used by shared stages. Each viz pipeline
// stores a *CommonState alongside the per-viz state and config in the
// type-keyed *pipeline.State.
type CommonState struct {
	// Inputs: set by the orchestrator before applying any stages.
	TargetPath         string
	Output             string
	Flags              *Flags
	RootConfig         *config.Config
	VizName            string // active visualization name for export trimming
	CLIFilters         []filter.Rule
	IncludeBinaryFiles bool // when true, retain binary files in the scanned tree

	// Populated by shared stages during the pipeline:
	FilterRules   []filter.Rule    // BuildFilterRules
	Requested     RequestedMetrics // viz-specific ResolveMetrics
	Root          *model.Directory // ScanFilesystem
	Width         int              // ResolveDimensions
	Height        int              // ResolveDimensions
	DrawingBounds geometry.Rect    // InitDrawingBounds + Reserve*Bounds
	Canvas        *canvas.Canvas   // viz-specific Render

	// Git history (populated by LoadGitHistory / GroupGitHistoryByFile / ExtractFileHistory).
	// GitHistory is written once and not mutated afterward; consumers may hold
	// *Commit references for the lifetime of CommonState.
	GitHistory    []git.Commit
	FileHistory   map[*model.File][]CommitRef
	FileTimeRange map[*model.File]TimeRange

	// Authorship history (populated by LoadAuthorHistory).
	// AuthorHistory holds per-file per-author contribution records, the
	// repo-wide last-active map, and the HEAD commit date used as the global
	// clock for all authorship window calculations.
	AuthorHistory git.AuthorHistoryResult
}
