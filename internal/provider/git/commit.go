package git

import (
	"io"
	"path/filepath"
	"time"

	gogit "github.com/go-git/go-git/v5"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// Signature mirrors go-git's object.Signature: an author or committer record
// captured at the moment a commit was made.
type Signature struct {
	Name  string
	Email string
	When  time.Time
}

// Commit is a single commit in the project history, carrying enough metadata
// for any downstream consumer (timeline, churn, authorship, message-mining).
// ChangedPaths is restricted to the tracked path set passed to BulkCommitHistory
// so the slice size stays bounded.
//
// Invariant: once BulkCommitHistory returns, no field of any returned Commit
// is mutated. Consumers may hold *Commit references (e.g. via CommitRef) for
// the lifetime of the slice.
type Commit struct {
	Hash         string
	Author       Signature
	Committer    Signature
	Message      string
	ParentHashes []string
	ChangedPaths []string // slash-separated, repo-relative
}

// CommitTotal returns the number of commits reachable from HEAD.
func CommitTotal(repoPath string) (int64, error) {
	return CommitTotalInRange(repoPath, time.Time{}, time.Time{})
}

// CommitTotalInRange returns the number of reachable commits within the supplied window.
func CommitTotalInRange(repoPath string, from, until time.Time) (int64, error) {
	s, err := getService(repoPath)
	if err != nil {
		return 0, eris.Wrap(err, "failed to open git repository")
	}

	return s.commitTotalInRange(from, until)
}

func (s *repoService) commitTotal() (int64, error) {
	return s.commitTotalInRange(time.Time{}, time.Time{})
}

func (s *repoService) commitTotalInRange(from, until time.Time) (int64, error) {
	s.repoMu.Lock()
	defer s.repoMu.Unlock()

	iter, err := s.commitIterator(from, until)
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	var total int64

	err = iter.ForEach(func(*object.Commit) error {
		total++

		return nil
	})
	if err != nil {
		return 0, eris.Wrap(err, "failed to iterate commits")
	}

	return total, nil
}

// BulkCommitHistory walks the commit graph once and returns one Commit per
// commit reachable from HEAD that touches at least one path in `tracked`.
// Commits that change no tracked path are omitted.
//
// onCommitProcessed is invoked after each commit is examined (including
// skipped ones), allowing callers to drive a progress meter.
func BulkCommitHistory(
	repoPath string,
	tracked map[string]bool,
	onCommitProcessed func(),
) ([]Commit, error) {
	return BulkCommitHistoryInRange(repoPath, tracked, time.Time{}, time.Time{}, onCommitProcessed)
}

// BulkCommitHistoryInRange filters traversed commits to the supplied date window.
func BulkCommitHistoryInRange(
	repoPath string,
	tracked map[string]bool,
	from, until time.Time,
	onCommitProcessed func(),
) ([]Commit, error) {
	s, err := getService(repoPath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to open git repository")
	}

	var commits []Commit

	err = s.walkTrackedHistoryInRange(tracked, from, until, onCommitProcessed, func(c *object.Commit, changed []trackedChange) {
		appendTrackedCommit(&commits, c, changed)
	})
	if err != nil {
		return nil, err
	}

	return commits, nil
}

// BulkCommitHistoryAndPrewarm walks the commit graph once, returning commits
// that touch tracked paths and prewarming requested file metric data.
func BulkCommitHistoryAndPrewarm(
	repoPath string,
	tracked map[string]bool,
	requested []metric.Name,
	onCommitProcessed func(),
) ([]Commit, error) {
	return BulkCommitHistoryAndPrewarmInRange(repoPath, tracked, requested, time.Time{}, time.Time{}, onCommitProcessed)
}

// BulkCommitHistoryAndPrewarmInRange filters traversed commits to the supplied date window.
func BulkCommitHistoryAndPrewarmInRange(
	repoPath string,
	tracked map[string]bool,
	requested []metric.Name,
	from, until time.Time,
	onCommitProcessed func(),
) ([]Commit, error) {
	s, err := getService(repoPath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to open git repository")
	}

	return s.bulkCommitHistoryAndPrewarmInRange(
		normalizeTrackedPaths(tracked),
		newMetricRequirements(requested),
		from,
		until,
		onCommitProcessed,
	)
}

func normalizeTrackedPaths(tracked map[string]bool) map[string]bool {
	for path := range tracked {
		if filepath.ToSlash(path) != path {
			normalized := make(map[string]bool, len(tracked))
			for path, included := range tracked {
				normalized[filepath.ToSlash(path)] = included
			}

			return normalized
		}
	}

	return tracked
}

func (s *repoService) bulkCommitHistoryAndPrewarmInRange(
	tracked map[string]bool,
	requirements metricRequirements,
	from, until time.Time,
	onCommitProcessed func(),
) ([]Commit, error) {
	cache := newBulkPrewarmCache(tracked, requirements)

	var commits []Commit

	err := s.walkTrackedHistoryInRange(tracked, from, until, onCommitProcessed, func(c *object.Commit, changed []trackedChange) {
		prewarmTrackedChanges(cache, c, changed, requirements)
		appendTrackedCommit(&commits, c, changed)
	})
	if err != nil {
		return nil, err
	}

	s.publishBulkPrewarmCache(cache, requirements)

	return commits, nil
}

func (s *repoService) commitIterator(from, until time.Time) (object.CommitIter, error) {
	head, err := s.repo.Head()
	if err != nil {
		return nil, eris.Wrap(err, "failed to get HEAD")
	}

	iter, err := s.repo.Log(&gogit.LogOptions{From: head.Hash()})
	if err != nil {
		return nil, eris.Wrap(err, "failed to start log iteration")
	}

	if !from.IsZero() || !until.IsZero() {
		filtered := &filteredCommitIter{iter: iter, from: from, until: until}
		return filtered, nil
	}

	return iter, nil
}

type filteredCommitIter struct {
	iter   object.CommitIter
	from   time.Time
	until  time.Time
	closed bool
}

func (f *filteredCommitIter) Next() (*object.Commit, error) {
	for {
		c, err := f.iter.Next()
		if err != nil {
			return nil, err
		}
		if !f.from.IsZero() && c.Author.When.Before(f.from) {
			continue
		}
		if !f.until.IsZero() && c.Author.When.After(f.until) {
			continue
		}

		return c, nil
	}
}

func (f *filteredCommitIter) ForEach(cb func(*object.Commit) error) error {
	for {
		c, err := f.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := cb(c); err != nil {
			return err
		}
	}
}

func (f *filteredCommitIter) Close() {
	if f.closed {
		return
	}
	f.closed = true
	f.iter.Close()
}

func newBulkPrewarmCache(
	tracked map[string]bool,
	requirements metricRequirements,
) map[string]*commitData {
	if len(requirements.processors) == 0 {
		return nil
	}

	cache := make(map[string]*commitData, len(tracked))
	for path := range tracked {
		cache[path] = &commitData{
			authors:      make(map[string]bool),
			hasLineStats: requirements.needsLineStats,
		}
	}

	return cache
}

func (s *repoService) publishBulkPrewarmCache(
	cache map[string]*commitData,
	requirements metricRequirements,
) {
	if cache != nil {
		s.mergeBulkPrewarmCache(cache, requirements)
	}
}

func prewarmTrackedChanges(
	cache map[string]*commitData,
	c *object.Commit,
	changed []trackedChange,
	requirements metricRequirements,
) {
	if cache == nil {
		return
	}

	for _, entry := range changed {
		data := cache[entry.path]
		if data == nil {
			continue
		}

		data.updateMetadata(c)

		if requirements.needsLineStats {
			data.updateChangeStats(entry.change)
		}
	}
}

func appendTrackedCommit(commits *[]Commit, c *object.Commit, changed []trackedChange) {
	if len(changed) == 0 {
		return
	}

	changedPaths := make([]string, 0, len(changed))
	for _, entry := range changed {
		changedPaths = append(changedPaths, entry.path)
	}

	*commits = append(*commits, Commit{
		Hash:         c.Hash.String(),
		Author:       toSignature(c.Author),
		Committer:    toSignature(c.Committer),
		Message:      c.Message,
		ParentHashes: parentHashes(c),
		ChangedPaths: changedPaths,
	})
}

func (s *repoService) walkTrackedHistoryInRange(
	tracked map[string]bool,
	from, until time.Time,
	onCommitProcessed func(),
	visit func(*object.Commit, []trackedChange),
) error {
	s.repoMu.Lock()
	defer s.repoMu.Unlock()

	iter, err := s.commitIterator(from, until)
	if err != nil {
		return err
	}
	defer iter.Close()

	err = iter.ForEach(func(c *object.Commit) error {
		changed := trackedChangesInCommit(c, tracked)

		if onCommitProcessed != nil {
			onCommitProcessed()
		}

		visit(c, changed)

		return nil
	})
	if err != nil {
		return eris.Wrap(err, "failed to iterate commits")
	}

	return nil
}

func toSignature(s object.Signature) Signature {
	return Signature{Name: s.Name, Email: s.Email, When: s.When}
}

func parentHashes(c *object.Commit) []string {
	if c.NumParents() == 0 {
		return nil
	}

	hashes := make([]string, 0, c.NumParents())
	for _, h := range c.ParentHashes {
		hashes = append(hashes, h.String())
	}

	return hashes
}
