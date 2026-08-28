package stages

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
)

// LoadAuthorHistory walks the commit graph once and populates
// c.AuthorHistory with per-file per-author contribution records,
// a repo-wide last-active map, and the HEAD commit date.
//
// This is the data foundation for all authorship metrics (#550).
// It must be called after ScanFilesystem (c.Root must be populated).
func LoadAuthorHistory(c *CommonState) error {
	repoRoot, err := git.RepoRootFor(c.Root.Path)
	if err != nil {
		return eris.Wrap(err, "failed to resolve git root")
	}

	tracked := buildTrackedPathSet(c.Root, repoRoot)

	total, err := git.CommitTotalInRange(repoRoot, c.Flags.From, c.Flags.Until)
	if err != nil {
		return eris.Wrap(err, "failed to count git commits")
	}

	onCommit, stop := BuildHistoryProgress(c.Flags, total)

	result, err := git.BulkAuthorHistoryInRange(repoRoot, tracked, false, c.Flags.From, c.Flags.Until, onCommit)

	stop()

	if err != nil {
		return eris.Wrap(err, "failed to load author history")
	}

	c.AuthorHistory = result

	return nil
}
