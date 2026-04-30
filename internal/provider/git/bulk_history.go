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
func changedInRootCommit(c *object.Commit, filePaths map[string]bool) []string {
	tree, err := c.Tree()
	if err != nil {
		return nil
	}

	var changed []string

	_ = tree.Files().ForEach(func(f *object.File) error {
		if filePaths[f.Name] {
			changed = append(changed, f.Name)
		}

		return nil
	})

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
		name := change.To.Name
		if name == "" {
			name = change.From.Name
		}

		if filePaths[name] {
			result = append(result, name)
		}
	}

	return result
}

// changedInMergeCommit returns tracked files that differ from ALL parents
// (not TREESAME to any parent). This matches git's history simplification.
func changedInMergeCommit(c *object.Commit, commitTree *object.Tree, filePaths map[string]bool) []string {
	// Collect files changed vs each parent
	parents := c.Parents()
	defer parents.Close()

	// Track which files differ from each parent
	diffFromParent := make([]map[string]bool, 0, c.NumParents())

	_ = parents.ForEach(func(parent *object.Commit) error {
		parentTree, err := parent.Tree()
		if err != nil {
			return nil //nolint:nilerr // skip parent on error
		}

		changes, err := object.DiffTree(parentTree, commitTree)
		if err != nil {
			return nil //nolint:nilerr // skip parent on error
		}

		diffs := make(map[string]bool, len(changes))
		for _, change := range changes {
			name := change.To.Name
			if name == "" {
				name = change.From.Name
			}

			if filePaths[name] {
				diffs[name] = true
			}
		}

		diffFromParent = append(diffFromParent, diffs)

		return nil
	})

	if len(diffFromParent) == 0 {
		return nil
	}

	// A file is modified only if it differs from ALL parents
	var result []string

	for path := range filePaths {
		differsFromAll := true

		for _, diffs := range diffFromParent {
			if !diffs[path] {
				differsFromAll = false

				break
			}
		}

		if differsFromAll {
			result = append(result, path)
		}
	}

	return result
}
