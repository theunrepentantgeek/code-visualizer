package git

import (
	"errors"
	"io"
	"iter"
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
func CommitTotalInRange(repoPath string, from time.Time, until time.Time) (int64, error) {
	s, err := getService(repoPath)
	if err != nil {
		return 0, eris.Wrap(err, "failed to open git repository")
	}

	return s.commitTotalInRange(from, until)
}

func (s *repoService) commitTotal() (int64, error) {
	return s.commitTotalInRange(time.Time{}, time.Time{})
}

func (s *repoService) commitTotalInRange(from time.Time, until time.Time) (int64, error) {
	s.repoMu.Lock()
	defer s.repoMu.Unlock()

	commits, err := s.commitIterator(from, until)
	if err != nil {
		return 0, err
	}

	var total int64

	for _, iterationErr := range commits {
		if iterationErr != nil {
			return 0, eris.Wrap(iterationErr, "failed to iterate commits")
		}

		total++
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
	from time.Time,
	until time.Time,
	onCommitProcessed func(),
) ([]Commit, error) {
	s, err := getService(repoPath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to open git repository")
	}

	var commits []Commit

	err = s.walkTrackedHistoryInRange(tracked, from, until, onCommitProcessed,
		func(c *object.Commit, changed []trackedChange) {
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
	from time.Time,
	until time.Time,
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
	from time.Time,
	until time.Time,
	onCommitProcessed func(),
) ([]Commit, error) {
	cache := newBulkPrewarmCache(tracked, requirements)

	var commits []Commit

	err := s.walkTrackedHistoryInRange(tracked, from, until, onCommitProcessed,
		func(c *object.Commit, changed []trackedChange) {
			prewarmTrackedChanges(cache, c, changed, requirements)
			appendTrackedCommit(&commits, c, changed)
		})
	if err != nil {
		return nil, err
	}

	s.publishBulkPrewarmCache(cache, requirements)

	return commits, nil
}

func (s *repoService) commitIterator(
	from time.Time,
	until time.Time,
) (iter.Seq2[*object.Commit, error], error) {
	head, err := s.repo.Head()
	if err != nil {
		return nil, eris.Wrap(err, "failed to get HEAD")
	}

	commitIter, err := s.repo.Log(&gogit.LogOptions{From: head.Hash()})
	if err != nil {
		return nil, eris.Wrap(err, "failed to start log iteration")
	}

	return filterCommitsInRange(commitSequence(commitIter), from, until), nil
}

func commitSequence(commitIter object.CommitIter) iter.Seq2[*object.Commit, error] {
	return func(yield func(*object.Commit, error) bool) {
		yieldCommits(commitIter, yield)
	}
}

func yieldCommits(commitIter object.CommitIter, yield func(*object.Commit, error) bool) {
	defer commitIter.Close()

	for {
		commit, done, iterationErr := nextCommit(commitIter)
		if done {
			return
		}

		if iterationErr != nil {
			yield(nil, iterationErr)

			return
		}

		if !yield(commit, nil) {
			return
		}
	}
}

func nextCommit(commitIter object.CommitIter) (*object.Commit, bool, error) {
	commit, err := commitIter.Next()
	if errors.Is(err, io.EOF) {
		return nil, true, nil
	}

	return commit, false, eris.Wrap(err, "failed to read commit")
}

func filterCommitsInRange(
	commits iter.Seq2[*object.Commit, error],
	from time.Time,
	until time.Time,
) iter.Seq2[*object.Commit, error] {
	return func(yield func(*object.Commit, error) bool) {
		for commit, iterationErr := range commits {
			if iterationErr != nil {
				yield(nil, iterationErr)

				return
			}

			if commitInRange(commit, from, until) && !yield(commit, nil) {
				return
			}
		}
	}
}

func commitInRange(commit *object.Commit, from time.Time, until time.Time) bool {
	return (from.IsZero() || !commit.Author.When.Before(from)) &&
		(until.IsZero() || !commit.Author.When.After(until))
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
	from time.Time,
	until time.Time,
	onCommitProcessed func(),
	visit func(*object.Commit, []trackedChange),
) error {
	s.repoMu.Lock()
	defer s.repoMu.Unlock()

	commits, err := s.commitIterator(from, until)
	if err != nil {
		return err
	}

	for c, iterationErr := range commits {
		if iterationErr != nil {
			return eris.Wrap(iterationErr, "failed to iterate commits")
		}

		changed := trackedChangesInCommit(c, tracked)

		if onCommitProcessed != nil {
			onCommitProcessed()
		}

		visit(c, changed)
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
