package goldentest

import (
	"time"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// buildSpiralHistory assigns each file a small, deterministic set of commits
// with pinned dates spread across a fixed window, and returns the FileHistory
// and FileTimeRange maps the spiral pipeline consumes. Pinned dates keep the
// time-bucketing reproducible.
func buildSpiralHistory(root *model.Directory) (
	map[*model.File][]stages.CommitRef,
	map[*model.File]stages.TimeRange,
) {
	history := make(map[*model.File][]stages.CommitRef)
	ranges := make(map[*model.File]stages.TimeRange)

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	var idx int

	model.WalkFiles(root, func(f *model.File) {
		// Two commits per file at deterministic offsets.
		first := base.AddDate(0, idx%6, 0)      // months 0..5
		second := first.AddDate(0, 0, 10+idx%5) // 10..14 days later

		c1 := &git.Commit{Hash: "c1-" + f.Path, Message: "create " + f.Name}
		c2 := &git.Commit{Hash: "c2-" + f.Path, Message: "update " + f.Name}

		history[f] = []stages.CommitRef{
			{Commit: c1, When: first},
			{Commit: c2, When: second},
		}
		ranges[f] = stages.TimeRange{Earliest: first, Latest: second}

		idx++
	})

	return history, ranges
}
