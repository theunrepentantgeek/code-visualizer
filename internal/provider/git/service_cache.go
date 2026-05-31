package git

import (
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rotisserie/eris"
)

// computeFileDiffStats computes the lines added and removed for a file in a
// non-root commit by diffing against the first parent. Returns (0, 0) for
// creation commits (file doesn't exist in parent).
func computeFileDiffStats(c *object.Commit, relPath string) (added, removed int64) {
	parent, err := c.Parent(0)
	if err != nil {
		return 0, 0
	}

	// Skip creation commits — file doesn't exist in parent.
	if _, hashErr := blobHash(parent, relPath); hashErr != nil {
		return 0, 0
	}

	parentTree, err := parent.Tree()
	if err != nil {
		return 0, 0
	}

	commitTree, err := c.Tree()
	if err != nil {
		return 0, 0
	}

	changes, err := object.DiffTree(parentTree, commitTree)
	if err != nil {
		return 0, 0
	}

	fileChanges := filterChangesForFile(changes, relPath)
	if len(fileChanges) == 0 {
		return 0, 0
	}

	patch, err := fileChanges.Patch()
	if err != nil {
		return 0, 0
	}

	for _, stat := range patch.Stats() {
		added += int64(stat.Addition)
		removed += int64(stat.Deletion)
	}

	return added, removed
}

// filterChangesForFile returns only the changes that affect the given file.
func filterChangesForFile(changes object.Changes, relPath string) object.Changes {
	for _, change := range changes {
		if changeName(change) == relPath {
			return object.Changes{change}
		}
	}

	return nil
}

// getCommitData returns cached commit data for the given file path, fetching it
// from git on first access. Concurrent requests for the same path are coalesced
// via singleflight so the git log is only read once per file per process run.
func (s *repoService) getCommitData(relPath string) (*commitData, error) {
	s.commitMu.RLock()

	if cached, ok := s.commitCache[relPath]; ok {
		s.commitMu.RUnlock()

		return cached, nil
	}

	s.commitMu.RUnlock()

	result, err, _ := s.commitGroup.Do(relPath, func() (any, error) {
		data, err := s.fetchCommitData(relPath)
		if err != nil {
			return nil, err
		}

		s.commitMu.Lock()
		s.commitCache[relPath] = data
		s.commitMu.Unlock()

		return data, nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "failed to get commit data")
	}

	cd, ok := result.(*commitData)
	if !ok {
		return nil, eris.New("unexpected commit cache result type")
	}

	return cd, nil
}

func (s *repoService) fetchCommitData(relPath string) (*commitData, error) {
	log, err := s.repo.Log(&gogit.LogOptions{FileName: &relPath})
	if err != nil {
		return nil, eris.Wrap(err, "failed to get git log")
	}
	defer log.Close()

	data := &commitData{
		authors: make(map[string]bool),
	}

	err = log.ForEach(func(c *object.Commit) error {
		processCommitForFile(c, relPath, data)

		return nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "failed to iterate commits")
	}

	return data, nil
}

// processCommitForFile updates commitData for a single commit that may or may
// not have modified the file. It checks TREESAME filtering, updates timestamps,
// author set, commit count, and diff stats.
func processCommitForFile(c *object.Commit, relPath string, data *commitData) {
	// go-git's FileName filter includes merge commits that didn't
	// actually modify the file. Skip those to avoid polluting
	// the newest timestamp with unrelated commit dates.
	if !commitModifiedFile(c, relPath) {
		return
	}

	when := c.Author.When
	if data.oldest.IsZero() || when.Before(data.oldest) {
		data.oldest = when
	}

	if data.newest.IsZero() || when.After(data.newest) {
		data.newest = when
	}

	data.authors[c.Author.Email] = true
	data.count++

	// Accumulate diff stats for non-root commits that modify an existing file.
	if c.NumParents() > 0 {
		added, removed := computeFileDiffStats(c, relPath)
		data.linesAdded += added
		data.linesRemoved += removed
	}
}

// commitModifiedFile returns true if the commit actually changed the file at
// relPath, as opposed to merely having it in the tree (which happens with merge
// commits). A commit modified the file only if it is NOT TREESAME to any parent,
// matching git's history simplification semantics. Specifically:
//   - root commits (no parents) are always considered as modifying the file,
//   - a commit is TREESAME to a parent when the file's blob hash is identical,
//   - a commit is "modified" only when it differs from ALL parents.
func commitModifiedFile(c *object.Commit, relPath string) bool {
	fileHash, err := blobHash(c, relPath)
	if err != nil {
		return true // conservative: include on error
	}

	parents := c.Parents()
	defer parents.Close()

	hasParent := false
	treesameToAny := false

	_ = parents.ForEach(func(parent *object.Commit) error {
		hasParent = true

		parentHash, hashErr := blobHash(parent, relPath)
		if hashErr == nil && parentHash == fileHash {
			treesameToAny = true
		}

		return nil
	})

	if !hasParent {
		return true // root commit — file was introduced
	}

	return !treesameToAny
}

// FileCommitTimestamps returns the author timestamps for all commits that modified
// the file at relPath, relative to the git worktree root discovered from repoPath.
// It uses the same TREESAME filtering as the metric providers.
func FileCommitTimestamps(repoPath, relPath string) ([]time.Time, error) {
	s, err := getService(repoPath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to open git repository")
	}

	return s.fetchCommitTimestamps(relPath)
}

func (s *repoService) fetchCommitTimestamps(relPath string) ([]time.Time, error) {
	log, err := s.repo.Log(&gogit.LogOptions{FileName: &relPath})
	if err != nil {
		return nil, eris.Wrap(err, "failed to get git log")
	}
	defer log.Close()

	var timestamps []time.Time

	err = log.ForEach(func(c *object.Commit) error {
		if !commitModifiedFile(c, relPath) {
			return nil
		}

		timestamps = append(timestamps, c.Author.When)

		return nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "failed to iterate commits")
	}

	return timestamps, nil
}

// blobHash returns the blob hash of the file at relPath within the commit's tree.
func blobHash(c *object.Commit, relPath string) (plumbing.Hash, error) {
	tree, err := c.Tree()
	if err != nil {
		return plumbing.ZeroHash, err //nolint:wrapcheck // internal helper
	}

	entry, err := tree.File(relPath)
	if err != nil {
		return plumbing.ZeroHash, err //nolint:wrapcheck // internal helper
	}

	return entry.Hash, nil
}
