package git

import (
	"strings"
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

	var commits []Commit

	err = iter.ForEach(func(c *object.Commit) error {
		changed := changedFilesInCommit(c, tracked)

		if onCommitProcessed != nil {
			onCommitProcessed()
		}

		if len(changed) == 0 {
			return nil
		}

		commits = append(commits, Commit{
			Hash:         c.Hash.String(),
			Author:       toSignature(c.Author),
			Committer:    toSignature(c.Committer),
			Message:      c.Message,
			ParentHashes: parentHashes(c),
			ChangedPaths: changed,
		})

		return nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "failed to iterate commits")
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
	s, err := getService(repoPath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to open git repository")
	}

	return s.bulkCommitHistoryAndPrewarm(
		normalizeTrackedPaths(tracked),
		newMetricRequirements(requested),
		onCommitProcessed,
	)
}

func normalizeTrackedPaths(tracked map[string]bool) map[string]bool {
	for path := range tracked {
		if strings.Contains(path, `\`) {
			normalized := make(map[string]bool, len(tracked))
			for path, included := range tracked {
				normalized[strings.ReplaceAll(path, `\`, `/`)] = included
			}

			return normalized
		}
	}

	return tracked
}

func (s *repoService) bulkCommitHistoryAndPrewarm(
	tracked map[string]bool,
	requirements metricRequirements,
	onCommitProcessed func(),
) ([]Commit, error) {
	iter, err := s.commitIterator()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	cache := newBulkPrewarmCache(tracked, requirements)

	var commits []Commit

	err = iter.ForEach(func(c *object.Commit) error {
		commit, include := processBulkCommit(
			c,
			tracked,
			cache,
			requirements,
			onCommitProcessed,
		)
		if include {
			commits = append(commits, commit)
		}

		return nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "failed to iterate commits")
	}

	s.publishBulkPrewarmCache(cache, requirements)

	return commits, nil
}

func (s *repoService) commitIterator() (object.CommitIter, error) {
	head, err := s.repo.Head()
	if err != nil {
		return nil, eris.Wrap(err, "failed to get HEAD")
	}

	iter, err := s.repo.Log(&gogit.LogOptions{From: head.Hash()})
	if err != nil {
		return nil, eris.Wrap(err, "failed to start log iteration")
	}

	return iter, nil
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

func processBulkCommit(
	c *object.Commit,
	tracked map[string]bool,
	cache map[string]*commitData,
	requirements metricRequirements,
	onCommitProcessed func(),
) (Commit, bool) {
	changed := trackedChangesInCommit(c, tracked)

	if onCommitProcessed != nil {
		onCommitProcessed()
	}

	prewarmTrackedChanges(cache, c, changed, requirements)

	if len(changed) == 0 {
		return Commit{}, false
	}

	return commitFromTrackedChanges(c, changed), true
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

func commitFromTrackedChanges(c *object.Commit, changed []trackedChange) Commit {
	changedPaths := make([]string, 0, len(changed))
	for _, entry := range changed {
		changedPaths = append(changedPaths, entry.path)
	}

	return Commit{
		Hash:         c.Hash.String(),
		Author:       toSignature(c.Author),
		Committer:    toSignature(c.Committer),
		Message:      c.Message,
		ParentHashes: parentHashes(c),
		ChangedPaths: changedPaths,
	}
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
