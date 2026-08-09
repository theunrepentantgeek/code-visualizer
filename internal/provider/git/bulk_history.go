package git

import (
	"time"

	gogit "github.com/go-git/go-git/v5"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rotisserie/eris"
)

// FileTimestamps maps relative file paths (slash-separated) to their commit timestamps.
type FileTimestamps map[string][]time.Time

type trackedChange struct {
	path   string
	change *object.Change
}

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
	trackedChanges := trackedChangesInCommit(c, filePaths)
	changed := make([]string, 0, len(trackedChanges))

	for _, entry := range trackedChanges {
		changed = append(changed, entry.path)
	}

	return changed
}

// trackedChangesInCommit returns the tracked paths modified by c along with
// their already-computed tree changes. For merge commits, a path must differ
// from all parents (TREESAME semantics).
func trackedChangesInCommit(c *object.Commit, filePaths map[string]bool) []trackedChange {
	if c.NumParents() == 0 {
		return trackedChangesInRootCommit(c, filePaths)
	}

	commitTree, err := c.Tree()
	if err != nil {
		return nil
	}

	if c.NumParents() == 1 {
		return trackedChangesVsParent(c, commitTree, filePaths)
	}

	return trackedChangesInMergeCommit(c, commitTree, filePaths)
}

// trackedChangesInRootCommit returns tracked files present in the root commit's tree.
// It iterates only the tracked file set (not all tree files) for efficiency in
// repos where filePaths is much smaller than the total number of tree files.
func trackedChangesInRootCommit(c *object.Commit, filePaths map[string]bool) []trackedChange {
	tree, err := c.Tree()
	if err != nil {
		return nil
	}

	changed := make([]trackedChange, 0, len(filePaths))

	for path := range filePaths {
		if _, err := tree.File(path); err == nil {
			changed = append(changed, trackedChange{path: path})
		}
	}

	return changed
}

// trackedChangesVsParent returns tracked files that differ between the commit and its
// single parent, using tree diff to efficiently skip unchanged subtrees.
func trackedChangesVsParent(
	c *object.Commit,
	commitTree *object.Tree,
	filePaths map[string]bool,
) []trackedChange {
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

	return trackedChangesFromDiff(changes, filePaths)
}

func trackedChangesFromDiff(changes object.Changes, filePaths map[string]bool) []trackedChange {
	result := make([]trackedChange, 0, len(changes))

	for i := range changes {
		change := changes[i]
		name := changeName(change)
		if filePaths[name] {
			result = append(result, trackedChange{path: name, change: change})
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

// trackedChangesInMergeCommit returns tracked files that differ from ALL parents
// (not TREESAME to any parent). This matches git's history simplification.
func trackedChangesInMergeCommit(
	c *object.Commit,
	commitTree *object.Tree,
	filePaths map[string]bool,
) []trackedChange {
	changesFromParent := collectParentChanges(c, commitTree, filePaths)
	if len(changesFromParent) == 0 {
		return nil
	}

	result := make([]trackedChange, 0, len(filePaths))

	for path := range filePaths {
		change, found := changesFromParent[0][path]
		if !found || !differsFromAllParents(path, changesFromParent) {
			continue
		}

		// Churn metrics historically diffed against the first parent.
		result = append(result, trackedChange{path: path, change: change})
	}

	return result
}

// collectParentChanges returns one diff-set per parent: the tracked files that
// differ between the parent and commitTree.
func collectParentChanges(
	c *object.Commit,
	commitTree *object.Tree,
	filePaths map[string]bool,
) []map[string]*object.Change {
	parents := c.Parents()
	defer parents.Close()

	result := make([]map[string]*object.Change, 0, c.NumParents())

	_ = parents.ForEach(func(parent *object.Commit) error {
		changes := trackedChangesFromParent(parent, commitTree, filePaths)
		if changes != nil {
			result = append(result, changes)
		}

		return nil
	})

	return result
}

// trackedChangesFromParent returns the tracked changes that differ between
// the parent commit's tree and commitTree.
func trackedChangesFromParent(
	parent *object.Commit,
	commitTree *object.Tree,
	filePaths map[string]bool,
) map[string]*object.Change {
	parentTree, err := parent.Tree()
	if err != nil {
		return nil
	}

	changes, err := object.DiffTree(parentTree, commitTree)
	if err != nil {
		return nil
	}

	diffs := make(map[string]*object.Change, len(changes))

	for i := range changes {
		change := changes[i]
		name := changeName(change)
		if filePaths[name] {
			diffs[name] = change
		}
	}

	return diffs
}

func differsFromAllParents(path string, changesFromParent []map[string]*object.Change) bool {
	for _, changes := range changesFromParent {
		if _, found := changes[path]; !found {
			return false
		}
	}

	return true
}
