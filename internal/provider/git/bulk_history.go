package git

import (
	"time"

	gogit "github.com/go-git/go-git/v5"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rotisserie/eris"
)

// FileTimestamps maps relative file paths (slash-separated) to their commit timestamps.
type FileTimestamps map[string][]time.Time

// BulkFileHistory walks the entire commit history once and returns the commit
// timestamps for each file in the provided set. This is dramatically faster than
// per-file log queries because it traverses the commit graph only once, using
// tree diffs to identify changed files per commit.
//
// For merge commits, a file is only considered modified if its blob differs from
// ALL parents (matching git's TREESAME simplification semantics).
//
// The optional onCommitProcessed callback is invoked after each commit is examined.
func BulkFileHistory(
	repoPath string,
	filePaths map[string]bool,
	onCommitProcessed func(),
) (FileTimestamps, error) {
	s, err := getService(repoPath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to open git repository")
	}

	head, err := s.repo.Head()
	if err != nil {
		return nil, eris.Wrap(err, "failed to get HEAD")
	}

	iter, err := s.repo.Log(&gogit.LogOptions{From: head.Hash()})
	if err != nil {
		return nil, eris.Wrap(err, "failed to start log iteration")
	}
	defer iter.Close()

	result := make(FileTimestamps)

	err = iter.ForEach(func(c *object.Commit) error {
		changed := changedFilesInCommit(c, filePaths)

		ts := c.Author.When
		for _, path := range changed {
			result[path] = append(result[path], ts)
		}

		if onCommitProcessed != nil {
			onCommitProcessed()
		}

		return nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "failed to iterate commits")
	}

	return result, nil
}

// changedFilesInCommit returns the subset of filePaths that were actually
// modified by this commit. For merge commits, a file must differ from ALL
// parents to be considered modified (TREESAME semantics).
func changedFilesInCommit(c *object.Commit, filePaths map[string]bool) []string {
	if c.NumParents() == 0 {
		return changedInRootCommit(c, filePaths)
	}

	commitTree, err := c.Tree()
	if err != nil {
		return nil
	}

	if c.NumParents() == 1 {
		return changedVsParent(c, commitTree, filePaths)
	}

	return changedInMergeCommit(c, commitTree, filePaths)
}

// changedInRootCommit returns tracked files present in the root commit's tree.
// It iterates only the tracked file set (not all tree files) for efficiency in
// repos where filePaths is much smaller than the total number of tree files.
func changedInRootCommit(c *object.Commit, filePaths map[string]bool) []string {
	tree, err := c.Tree()
	if err != nil {
		return nil
	}

	changed := make([]string, 0, len(filePaths))

	for path := range filePaths {
		if _, err := tree.File(path); err == nil {
			changed = append(changed, path)
		}
	}

	return changed
}

// changedVsParent returns tracked files that differ between the commit and its
// single parent, using tree diff to efficiently skip unchanged subtrees.
func changedVsParent(c *object.Commit, commitTree *object.Tree, filePaths map[string]bool) []string {
	parent, err := c.Parent(0)
	if err != nil {
		return nil
	}

	parentTree, err := parent.Tree()
	if err != nil {
		return nil
	}

	changes, err := object.DiffTree(parentTree, commitTree)
	if err != nil {
		return nil
	}

	var result []string

	for _, change := range changes {
		name := changeName(change)
		if filePaths[name] {
			result = append(result, name)
		}
	}

	return result
}

// changeName returns the file path affected by a tree change entry.
func changeName(change *object.Change) string {
	if change.To.Name != "" {
		return change.To.Name
	}

	return change.From.Name
}

// changedInMergeCommit returns tracked files that differ from ALL parents
// (not TREESAME to any parent). This matches git's history simplification.
func changedInMergeCommit(c *object.Commit, commitTree *object.Tree, filePaths map[string]bool) []string {
	diffFromParent := collectParentDiffs(c, commitTree, filePaths)
	if len(diffFromParent) == 0 {
		return nil
	}

	return filesChangedVsAllParents(filePaths, diffFromParent)
}

// collectParentDiffs returns one diff-set per parent: the tracked files that
// differ between the parent and commitTree.
func collectParentDiffs(c *object.Commit, commitTree *object.Tree, filePaths map[string]bool) []map[string]bool {
	parents := c.Parents()
	defer parents.Close()

	result := make([]map[string]bool, 0, c.NumParents())

	_ = parents.ForEach(func(parent *object.Commit) error {
		diffs := diffTrackedFiles(parent, commitTree, filePaths)
		if diffs != nil {
			result = append(result, diffs)
		}

		return nil
	})

	return result
}

// diffTrackedFiles returns the set of tracked files that differ between
// the parent commit's tree and commitTree.
func diffTrackedFiles(parent *object.Commit, commitTree *object.Tree, filePaths map[string]bool) map[string]bool {
	parentTree, err := parent.Tree()
	if err != nil {
		return nil
	}

	changes, err := object.DiffTree(parentTree, commitTree)
	if err != nil {
		return nil
	}

	diffs := make(map[string]bool, len(changes))

	for _, change := range changes {
		name := changeName(change)
		if filePaths[name] {
			diffs[name] = true
		}
	}

	return diffs
}

// filesChangedVsAllParents returns files that differ from every parent.
func filesChangedVsAllParents(filePaths map[string]bool, diffFromParent []map[string]bool) []string {
	var result []string

	for path := range filePaths {
		if differsFromAllParents(path, diffFromParent) {
			result = append(result, path)
		}
	}

	return result
}

func differsFromAllParents(path string, diffFromParent []map[string]bool) bool {
	for _, diffs := range diffFromParent {
		if !diffs[path] {
			return false
		}
	}

	return true
}
