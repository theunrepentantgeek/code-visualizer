package git

import (
	"maps"
	"path/filepath"
	"time"

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
	return CommitTotalInHistoryRange(repoPath, HistoryRange{})
}

// CommitTotalInRange returns the number of reachable commits within the supplied window.
func CommitTotalInRange(repoPath string, from time.Time, until time.Time) (int64, error) {
	return CommitTotalInHistoryRange(repoPath, HistoryRange{From: from, Until: until})
}

// CommitTotalInHistoryRange returns the number of commits selected by historyRange.
func CommitTotalInHistoryRange(repoPath string, historyRange HistoryRange) (int64, error) {
	s, err := getService(repoPath)
	if err != nil {
		return 0, eris.Wrap(err, "failed to open git repository")
	}

	return s.commitTotalInHistoryRange(historyRange)
}

func (s *repoService) commitTotal() (int64, error) {
	return s.commitTotalInHistoryRange(HistoryRange{})
}

func (s *repoService) commitTotalInHistoryRange(historyRange HistoryRange) (int64, error) {
	s.repoMu.Lock()
	defer s.repoMu.Unlock()

	commits, err := s.commitIterator(historyRange)
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
	return BulkCommitHistoryInHistoryRange(repoPath, tracked, HistoryRange{}, onCommitProcessed)
}

// BulkCommitHistoryInRange filters traversed commits to the supplied date window.
func BulkCommitHistoryInRange(
	repoPath string,
	tracked map[string]bool,
	from time.Time,
	until time.Time,
	onCommitProcessed func(),
) ([]Commit, error) {
	return BulkCommitHistoryInHistoryRange(
		repoPath,
		tracked,
		HistoryRange{From: from, Until: until},
		onCommitProcessed,
	)
}

// BulkCommitHistoryInHistoryRange filters traversed commits to historyRange.
func BulkCommitHistoryInHistoryRange(
	repoPath string,
	tracked map[string]bool,
	historyRange HistoryRange,
	onCommitProcessed func(),
) ([]Commit, error) {
	s, err := getService(repoPath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to open git repository")
	}

	var commits []Commit

	err = s.walkTrackedHistoryInHistoryRange(tracked, historyRange, onCommitProcessed,
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
	return BulkCommitHistoryAndPrewarmInHistoryRange(
		repoPath,
		tracked,
		requested,
		HistoryRange{},
		onCommitProcessed,
	)
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
	return BulkCommitHistoryAndPrewarmInHistoryRange(
		repoPath,
		tracked,
		requested,
		HistoryRange{From: from, Until: until},
		onCommitProcessed,
	)
}

// BulkCommitHistoryAndPrewarmInHistoryRange filters commits to historyRange.
func BulkCommitHistoryAndPrewarmInHistoryRange(
	repoPath string,
	tracked map[string]bool,
	requested []metric.Name,
	historyRange HistoryRange,
	onCommitProcessed func(),
) ([]Commit, error) {
	s, err := getService(repoPath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to open git repository")
	}

	return s.bulkCommitHistoryAndPrewarmInHistoryRange(
		normalizeTrackedPaths(tracked),
		newMetricRequirements(requested),
		historyRange,
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

func (s *repoService) bulkCommitHistoryAndPrewarmInHistoryRange(
	tracked map[string]bool,
	requirements metricRequirements,
	historyRange HistoryRange,
	onCommitProcessed func(),
) ([]Commit, error) {
	cache := newBulkPrewarmCache(tracked, requirements)

	var commits []Commit

	err := s.walkTrackedHistoryInHistoryRange(tracked, historyRange, onCommitProcessed,
		func(c *object.Commit, changed []trackedChange) {
			prewarmTrackedChanges(cache, c, changed, requirements)
			appendTrackedCommit(&commits, c, changed)
		})
	if err != nil {
		return nil, err
	}

	s.replaceBulkPrewarmCache(cache)

	return commits, nil
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

func (s *repoService) replaceBulkPrewarmCache(cache map[string]*commitData) {
	if cache == nil {
		return
	}

	s.commitMu.Lock()
	defer s.commitMu.Unlock()

	maps.Copy(s.commitCache, cache)
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

func (s *repoService) walkTrackedHistoryInHistoryRange(
	tracked map[string]bool,
	historyRange HistoryRange,
	onCommitProcessed func(),
	visit func(*object.Commit, []trackedChange),
) error {
	s.repoMu.Lock()
	defer s.repoMu.Unlock()

	commits, err := s.commitIterator(historyRange)
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
