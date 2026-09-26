package git

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
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
// Changes is restricted to the tracked path set passed to BulkCommitHistory
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
	Changes      []FileChange
}

// FileChange records line statistics for one tracked path in a commit.
type FileChange struct {
	Path         string
	LinesAdded   int64
	LinesRemoved int64
}

type historyChangeCacheKey struct {
	commit string
	paths  string
}

type historyChangeMode int

const (
	loadTrackedChanges historyChangeMode = iota
	loadCachedChangeStats
)

// CommitTotal returns the number of commits reachable from HEAD.
func CommitTotal(repoPath string) (int64, error) {
	return CommitTotalInHistoryRange(context.Background(), repoPath, HistoryRange{})
}

// CommitTotalInHistoryRange returns the number of commits selected by historyRange.
func CommitTotalInHistoryRange(
	ctx context.Context,
	repoPath string,
	historyRange HistoryRange,
) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, eris.Wrap(err, "commit counting cancelled")
	}

	s, err := getService(repoPath)
	if err != nil {
		return 0, eris.Wrap(err, "failed to open git repository")
	}

	return s.commitTotalInHistoryRange(ctx, historyRange)
}

func (s *repoService) commitTotalInHistoryRange(
	ctx context.Context,
	historyRange HistoryRange,
) (int64, error) {
	s.repoMu.Lock()
	defer s.repoMu.Unlock()

	commits, err := s.commitIterator(ctx, historyRange)
	if err != nil {
		return 0, err
	}

	var total int64

	for _, iterationErr := range commits {
		if err := ctx.Err(); err != nil {
			return 0, eris.Wrap(err, "commit counting cancelled")
		}

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
	return BulkCommitHistoryInHistoryRange(
		context.Background(),
		repoPath,
		tracked,
		HistoryRange{},
		onCommitProcessed,
	)
}

// BulkCommitHistoryInHistoryRange filters traversed commits to historyRange.
func BulkCommitHistoryInHistoryRange(
	ctx context.Context,
	repoPath string,
	tracked map[string]bool,
	historyRange HistoryRange,
	onCommitProcessed func(),
) ([]Commit, error) {
	if err := ctx.Err(); err != nil {
		return nil, eris.Wrap(err, "commit history loading cancelled")
	}

	s, err := getService(repoPath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to open git repository")
	}

	var commits []Commit

	err = s.walkTrackedHistoryInHistoryRange(ctx, tracked, historyRange, loadTrackedChanges, onCommitProcessed,
		func(c *object.Commit, changed []trackedChange) {
			appendTrackedCommit(&commits, c, changed, metricRequirements{})
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
		context.Background(),
		repoPath,
		tracked,
		requested,
		HistoryRange{},
		onCommitProcessed,
	)
}

// BulkCommitHistoryAndPrewarmInHistoryRange filters commits to historyRange.
func BulkCommitHistoryAndPrewarmInHistoryRange(
	ctx context.Context,
	repoPath string,
	tracked map[string]bool,
	requested []metric.Name,
	historyRange HistoryRange,
	onCommitProcessed func(),
) ([]Commit, error) {
	if err := ctx.Err(); err != nil {
		return nil, eris.Wrap(err, "commit history prewarming cancelled")
	}

	s, err := getService(repoPath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to open git repository")
	}

	return s.bulkCommitHistoryAndPrewarmInHistoryRange(
		ctx,
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
	ctx context.Context,
	tracked map[string]bool,
	requirements metricRequirements,
	historyRange HistoryRange,
	onCommitProcessed func(),
) ([]Commit, error) {
	cache := newBulkPrewarmCache(tracked, requirements)

	var commits []Commit

	changeMode := loadTrackedChanges
	if requirements.needsCommitStats || requirements.needsLineStats {
		changeMode = loadCachedChangeStats
	}

	err := s.walkTrackedHistoryInHistoryRange(ctx, tracked, historyRange, changeMode, onCommitProcessed,
		func(c *object.Commit, changed []trackedChange) {
			prewarmTrackedChanges(cache, c, changed, requirements)
			appendTrackedCommit(&commits, c, changed, requirements)
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

	s.commitCache = cache
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
			if entry.statsLoaded {
				data.addChangeStats(entry)
			} else {
				data.updateChangeStats(entry.change)
			}
		}
	}
}

func appendTrackedCommit(
	commits *[]Commit,
	c *object.Commit,
	changed []trackedChange,
	requirements metricRequirements,
) {
	if len(changed) == 0 {
		return
	}

	changes := make([]FileChange, 0, len(changed))
	for _, entry := range changed {
		change := FileChange{Path: entry.path}

		if requirements.needsCommitStats {
			if entry.statsLoaded {
				change.LinesAdded = entry.linesAdded
				change.LinesRemoved = entry.linesRemoved
			} else {
				data := &commitData{}
				data.updateChangeStats(entry.change)
				change.LinesAdded = data.linesAdded
				change.LinesRemoved = data.linesRemoved
			}
		}

		changes = append(changes, change)
	}

	*commits = append(*commits, Commit{
		Hash:         c.Hash.String(),
		Author:       toSignature(c.Author),
		Committer:    toSignature(c.Committer),
		Message:      c.Message,
		ParentHashes: parentHashes(c),
		Changes:      changes,
	})
}

//nolint:revive // The history walk keeps lock, cache, filtering, and visitation in one transaction.
func (s *repoService) walkTrackedHistoryInHistoryRange(
	ctx context.Context,
	tracked map[string]bool,
	historyRange HistoryRange,
	changeMode historyChangeMode,
	onCommitProcessed func(),
	visit func(*object.Commit, []trackedChange),
) error {
	s.repoMu.Lock()
	defer s.repoMu.Unlock()

	commits, err := s.commitIterator(ctx, historyRange)
	if err != nil {
		return err
	}

	cacheKey := trackedPathsCacheKey(tracked)

	for c, iterationErr := range commits {
		if err := ctx.Err(); err != nil {
			return eris.Wrap(err, "tracked history loading cancelled")
		}

		if iterationErr != nil {
			return eris.Wrap(iterationErr, "failed to iterate commits")
		}

		var changed []trackedChange
		if changeMode == loadCachedChangeStats {
			changed = s.cachedTrackedChanges(c, tracked, cacheKey)
		} else {
			changed = trackedChangesInCommit(c, tracked)
		}

		if onCommitProcessed != nil {
			onCommitProcessed()
		}

		if err := ctx.Err(); err != nil {
			return eris.Wrap(err, "tracked history loading cancelled")
		}

		visit(c, changed)
	}

	return nil
}

func (s *repoService) cachedTrackedChanges(
	commit *object.Commit,
	tracked map[string]bool,
	pathsKey string,
) []trackedChange {
	if s == nil || commit == nil {
		return nil
	}

	if s.historyChangeCache == nil {
		s.historyChangeCache = make(map[historyChangeCacheKey][]trackedChange)
	}

	key := historyChangeCacheKey{commit: commit.Hash.String(), paths: pathsKey}

	changes, ok := s.historyChangeCache[key]
	if ok {
		return changes
	}

	changes = trackedChangesInCommit(commit, tracked)
	for index := range changes {
		change := &changes[index]
		data := &commitData{}
		data.updateChangeStats(change.change)
		change.change = nil
		change.linesAdded = data.linesAdded
		change.linesRemoved = data.linesRemoved
		change.statsLoaded = true
	}

	s.historyChangeCache[key] = changes

	return changes
}

func trackedPathsCacheKey(tracked map[string]bool) string {
	paths := make([]string, 0, len(tracked))
	for path, included := range tracked {
		if included {
			paths = append(paths, path)
		}
	}

	slices.Sort(paths)

	return strings.Join(paths, "\x00")
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
