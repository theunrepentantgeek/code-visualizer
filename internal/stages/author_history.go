package stages

import (
	"context"
	"errors"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/progress"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
)

// LoadAuthorHistory walks the commit graph once and populates
// c.AuthorHistory with per-file per-author contribution records,
// a repo-wide last-active map, and the HEAD commit date.
//
// This is the data foundation for all authorship metrics (#550).
// It must be called after ScanFilesystem (c.Root must be populated).
func LoadAuthorHistory(c *CommonState, ctx context.Context, sink progress.Sink) error {
	repoRoot, err := repoRootForState(c, "author history")
	if err != nil {
		return eris.Wrap(err, "failed to resolve git root")
	}

	tracked := buildTrackedPathSet(c.Root, repoRoot)

	historyRange := c.Flags.HistoryRange

	total, err := git.CommitTotalInHistoryRange(ctx, repoRoot, historyRange)
	if err != nil {
		if errors.Is(err, ctx.Err()) {
			return ctx.Err()
		}

		return eris.Wrap(err, "failed to count git commits")
	}

	historyProg := newHistoryProgress(sink, total)

	result, err := git.BulkAuthorHistoryInHistoryRange(
		ctx,
		repoRoot,
		tracked,
		false,
		historyRange,
		historyProg.OnCommit,
	)

	if err != nil {
		if errors.Is(err, ctx.Err()) {
			return ctx.Err()
		}

		return eris.Wrap(err, "failed to load author history")
	}
	if err := historyProg.Err(); err != nil {
		return eris.Wrap(err, "report author progress")
	}

	c.AuthorHistory = result

	return nil
}
