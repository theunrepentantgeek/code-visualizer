package stages

import (
	"path/filepath"
	"time"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
)

// CommitRef points back into CommonState.GitHistory with the per-file
// when-touched timestamp. Storing a pointer avoids duplicating Author /
// Message / ParentHashes per file-commit pair.
type CommitRef struct {
	Commit *git.Commit
	When   time.Time
}

// TimeRange is the earliest and latest commit times observed for a file.
type TimeRange struct {
	Earliest time.Time
	Latest   time.Time
}

// LoadGitHistory walks the commit graph once and populates c.GitHistory.
// It returns an error when no commits touch any tracked file — visualizations
// that depend on git history cannot proceed in that case.
func LoadGitHistory(c *CommonState) error {
	repoRoot, err := git.RepoRootFor(c.Root.Path)
	if err != nil {
		return eris.Wrap(err, "failed to resolve git root")
	}

	tracked := buildTrackedPathSet(c.Root, repoRoot)

	onCommit, stop := BuildHistoryProgress(c.Flags)

	commits, err := git.BulkCommitHistory(repoRoot, tracked, onCommit)

	stop()

	if err != nil {
		return eris.Wrap(err, "failed to load commit history")
	}

	if len(commits) == 0 {
		return eris.New("no commit history found; commit-history-dependent visualizations require git commits")
	}

	c.GitHistory = commits

	return nil
}

// GroupGitHistoryByFile joins c.GitHistory against c.Root and writes
// c.FileHistory: each file maps to the CommitRefs that touched it.
func GroupGitHistoryByFile(c *CommonState) error {
	repoRoot, err := git.RepoRootFor(c.Root.Path)
	if err != nil {
		return eris.Wrap(err, "failed to resolve git root")
	}

	byPath := indexFilesByRepoRelativePath(c.Root, repoRoot)

	result := make(map[*model.File][]CommitRef)

	for i := range c.GitHistory {
		commit := &c.GitHistory[i]

		for _, path := range commit.ChangedPaths {
			file, ok := byPath[path]
			if !ok {
				continue
			}

			result[file] = append(result[file], CommitRef{
				Commit: commit,
				When:   commit.Author.When,
			})
		}
	}

	c.FileHistory = result

	return nil
}

// ExtractFileHistory folds c.FileHistory into per-file earliest/latest
// timestamps and writes c.FileTimeRange.
func ExtractFileHistory(c *CommonState) error {
	result := make(map[*model.File]TimeRange, len(c.FileHistory))

	for file, refs := range c.FileHistory {
		if len(refs) == 0 {
			continue
		}

		result[file] = foldCommitRefs(refs)
	}

	c.FileTimeRange = result

	return nil
}

// PruneFileHistoryToTree drops FileHistory / FileTimeRange entries whose file is
// no longer present in c.Root. Git history is loaded against the unfiltered
// scan, so once a tree-pruning stage (e.g. FilterBinaryFiles) removes files,
// their history must be dropped too — otherwise excluded files would still
// influence the global time range, time-bucket membership, and per-bucket
// aggregations.
func PruneFileHistoryToTree(c *CommonState) error {
	if len(c.FileHistory) == 0 && len(c.FileTimeRange) == 0 {
		return nil
	}

	surviving := make(map[*model.File]struct{})

	model.WalkFiles(c.Root, func(f *model.File) {
		surviving[f] = struct{}{}
	})

	for file := range c.FileHistory {
		if _, ok := surviving[file]; !ok {
			delete(c.FileHistory, file)
		}
	}

	for file := range c.FileTimeRange {
		if _, ok := surviving[file]; !ok {
			delete(c.FileTimeRange, file)
		}
	}

	return nil
}

func foldCommitRefs(refs []CommitRef) TimeRange {
	earliest := refs[0].When
	latest := refs[0].When

	for _, r := range refs[1:] {
		if r.When.Before(earliest) {
			earliest = r.When
		}

		if r.When.After(latest) {
			latest = r.When
		}
	}

	return TimeRange{Earliest: earliest, Latest: latest}
}

// CommitTimeRange folds the per-file ranges in Common().FileTimeRange into a
// single global earliest/latest pair. Returns the zero TimeRange when the map
// is empty.
func CommitTimeRange(fileRanges map[*model.File]TimeRange) TimeRange {
	var (
		set      bool
		earliest time.Time
		latest   time.Time
	)

	for _, r := range fileRanges {
		if !set {
			earliest = r.Earliest
			latest = r.Latest
			set = true

			continue
		}

		if r.Earliest.Before(earliest) {
			earliest = r.Earliest
		}

		if r.Latest.After(latest) {
			latest = r.Latest
		}
	}

	return TimeRange{Earliest: earliest, Latest: latest}
}

func buildTrackedPathSet(root *model.Directory, repoRoot string) map[string]bool {
	tracked := make(map[string]bool)

	model.WalkFiles(root, func(f *model.File) {
		rel, err := filepath.Rel(repoRoot, f.Path)
		if err != nil {
			return
		}

		tracked[filepath.ToSlash(rel)] = true
	})

	return tracked
}

func indexFilesByRepoRelativePath(root *model.Directory, repoRoot string) map[string]*model.File {
	index := make(map[string]*model.File)

	model.WalkFiles(root, func(f *model.File) {
		rel, err := filepath.Rel(repoRoot, f.Path)
		if err != nil {
			return
		}

		index[filepath.ToSlash(rel)] = f
	})

	return index
}
