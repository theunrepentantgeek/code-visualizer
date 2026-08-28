// Package git provides metric providers for git-derived metrics.
package git

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	gogit "github.com/go-git/go-git/v5"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rotisserie/eris"
	"golang.org/x/sync/singleflight"
)

type repoService struct {
	repo              *gogit.Repository
	rootPath          string // git worktree root (absolute path)
	repoMu            sync.Mutex
	commitGroup       singleflight.Group
	commitMu          sync.RWMutex
	commitCache       map[string]*commitData
	fetchCommitDataFn commitDataFetcher
	bulkGroup         singleflight.Group
}

// RepoRoot returns the absolute path to the git worktree root.
func (s *repoService) RepoRoot() string {
	return s.rootPath
}

var (
	servicesMu sync.Mutex
	services   = make(map[string]*serviceResult)
)

type serviceResult struct {
	svc *repoService
	err error
}

func getService(repoPath string) (*repoService, error) {
	servicesMu.Lock()
	defer servicesMu.Unlock()

	if result, exists := services[repoPath]; exists {
		return result.svc, result.err
	}

	repo, err := gogit.PlainOpenWithOptions(repoPath, &gogit.PlainOpenOptions{
		DetectDotGit:          true,
		EnableDotGitCommonDir: true,
	})
	if err != nil {
		err = eris.Wrap(err, "failed to open git repository")
		services[repoPath] = &serviceResult{nil, err}

		return nil, err
	}

	rootPath := repoPath

	wt, err := repo.Worktree()
	if err == nil {
		rootPath = wt.Filesystem.Root()
	} else if !errors.Is(err, gogit.ErrIsBareRepository) {
		err = eris.Wrap(err, "failed to get git worktree")
		services[repoPath] = &serviceResult{nil, err}

		return nil, err
	}

	if result, exists := services[rootPath]; exists {
		services[repoPath] = result

		return result.svc, result.err
	}

	svc := &repoService{
		repo:        repo,
		rootPath:    rootPath,
		commitCache: make(map[string]*commitData),
	}
	result := &serviceResult{svc, nil}
	services[repoPath] = result
	services[rootPath] = result

	return svc, nil
}

// resetService clears the cached service. Test use only.
func resetService() {
	servicesMu.Lock()
	defer servicesMu.Unlock()

	services = make(map[string]*serviceResult)
}

var errUntracked = errors.New("file has no git history")

func (s *repoService) anyPathHasGitHistory(paths map[string]bool) bool {
	s.commitMu.RLock()
	defer s.commitMu.RUnlock()

	for path := range paths {
		if data, ok := s.commitCache[path]; ok && data.count > 0 {
			return true
		}
	}

	return false
}

type commitDataFetcher func(string) (*commitData, error)

// metricFor is a generic helper that fetches commit data for a file, checks
// whether the file is untracked (count == 0), and applies the compute function.
func metricFor[T int64 | float64](
	relPath string,
	fetch commitDataFetcher,
	compute func(*commitData) T,
) (T, error) {
	data, err := fetch(relPath)
	if err != nil {
		var zero T

		return zero, err
	}

	if data.count == 0 {
		var zero T

		return zero, errUntracked
	}

	return compute(data), nil
}

func (s *repoService) fileAge(relPath string) (int64, error) {
	return metricFor(relPath, s.getMetadataCommitData, func(data *commitData) int64 {
		return int64(time.Since(data.oldest).Hours() / 24)
	})
}

func (s *repoService) fileFreshness(relPath string) (int64, error) {
	return metricFor(relPath, s.getMetadataCommitData, func(data *commitData) int64 {
		return int64(time.Since(data.newest).Hours() / 24)
	})
}

func (s *repoService) authorCount(relPath string) (int64, error) {
	return metricFor(relPath, s.getMetadataCommitData, func(data *commitData) int64 {
		return int64(len(data.authors))
	})
}

func (s *repoService) commitCount(relPath string) (int64, error) {
	return metricFor(relPath, s.getMetadataCommitData, func(data *commitData) int64 {
		return data.count
	})
}

func (s *repoService) totalLinesAdded(relPath string) (int64, error) {
	return metricFor(relPath, s.getLineStatsCommitData, func(data *commitData) int64 {
		return data.linesAdded
	})
}

func (s *repoService) totalLinesRemoved(relPath string) (int64, error) {
	return metricFor(relPath, s.getLineStatsCommitData, func(data *commitData) int64 {
		return data.linesRemoved
	})
}

const monthHours = 24 * 30.44

func (s *repoService) commitDensity(relPath string) (float64, error) {
	return metricFor(relPath, s.getMetadataCommitData, func(data *commitData) float64 {
		fileAgeMonths := time.Since(data.oldest).Hours() / monthHours
		if fileAgeMonths < 1 {
			fileAgeMonths = 1
		}

		return float64(data.count) / fileAgeMonths
	})
}

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

type commitDataCacheLookup func(string) *commitData

func (s *repoService) getMetadataCommitData(relPath string) (*commitData, error) {
	return s.getCommitData(relPath, "metadata", s.cachedCommitData)
}

func (s *repoService) getLineStatsCommitData(relPath string) (*commitData, error) {
	return s.getCommitData(relPath, "line-stats", s.cachedLineStatsCommitData)
}

// getCommitData returns cached commit data for the given file path, fetching it
// from git on first access. Concurrent requests with the same completeness
// requirement are coalesced via singleflight.
func (s *repoService) getCommitData(
	relPath string,
	completeness string,
	cacheLookup commitDataCacheLookup,
) (*commitData, error) {
	if cached := cacheLookup(relPath); cached != nil {
		return cached, nil
	}

	result, err, _ := s.commitGroup.Do(completeness+":"+relPath, func() (any, error) {
		return s.fetchAndCacheCommitData(relPath, cacheLookup)
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

func (s *repoService) cachedCommitData(relPath string) *commitData {
	s.commitMu.RLock()
	defer s.commitMu.RUnlock()

	return s.commitCache[relPath]
}

func (s *repoService) cachedLineStatsCommitData(relPath string) *commitData {
	cached := s.cachedCommitData(relPath)
	if cached == nil {
		return nil
	}

	if !cached.hasLineStats {
		return nil
	}

	return cached
}

func (s *repoService) fetchAndCacheCommitData(
	relPath string,
	cacheLookup commitDataCacheLookup,
) (*commitData, error) {
	if cached := cacheLookup(relPath); cached != nil {
		return cached, nil
	}

	fetch := s.fetchCommitData
	if s.fetchCommitDataFn != nil {
		fetch = s.fetchCommitDataFn
	}

	data, err := fetch(relPath)
	if err != nil {
		return nil, err
	}

	return s.cacheFetchedCommitData(relPath, data), nil
}

func (s *repoService) cacheFetchedCommitData(relPath string, data *commitData) *commitData {
	s.commitMu.Lock()
	defer s.commitMu.Unlock()

	if cached := s.commitCache[relPath]; cached != nil && cached.hasLineStats {
		return cached
	}

	s.commitCache[relPath] = data

	return data
}

func (s *repoService) fetchCommitData(relPath string) (*commitData, error) {
	s.repoMu.Lock()
	defer s.repoMu.Unlock()

	log, err := s.repo.Log(&gogit.LogOptions{FileName: &relPath})
	if err != nil {
		return nil, eris.Wrap(err, "failed to get git log")
	}
	defer log.Close()

	data := &commitData{
		authors:      make(map[string]bool),
		hasLineStats: true,
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

	data.updateMetadata(c)

	// Accumulate diff stats for non-root commits that modify an existing file.
	if c.NumParents() > 0 {
		updateLineStatsForFile(data, c, relPath)
	}
}

func updateLineStatsForFile(data *commitData, c *object.Commit, relPath string) {
	added, removed := computeFileDiffStats(c, relPath)
	data.linesAdded += added
	data.linesRemoved += removed
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

// RepoRootFor returns the git worktree root for the given path.
func RepoRootFor(repoPath string) (string, error) {
	s, err := getService(repoPath)
	if err != nil {
		return "", eris.Wrap(err, "failed to open git repository")
	}

	return s.RepoRoot(), nil
}

func (s *repoService) fetchCommitTimestamps(relPath string) ([]time.Time, error) {
	s.repoMu.Lock()
	defer s.repoMu.Unlock()

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

// bulkPrewarm pre-populates the commit cache for all provided file paths by
// walking the commit history once. This is dramatically faster than the default
// per-file path when many files share the same repository — e.g. 193 files
// require ~193 s with per-file git log; bulkPrewarm does it in one pass.
//
// If any paths are already cached, they are skipped unless a churn request
// needs line stats the cache does not contain. The function is safe for
// concurrent use; concurrent calls are coalesced via a singleflight group.
func (s *repoService) bulkPrewarm(
	paths map[string]bool,
	requirements metricRequirements,
	onCommitProcessed func(),
) error {
	missing, groupKey := s.bulkPrewarmWork(paths, requirements)
	if len(missing) == 0 {
		return nil
	}

	_, err, _ := s.bulkGroup.Do(groupKey, func() (any, error) {
		missing, _ = s.bulkPrewarmWork(paths, requirements)
		if len(missing) == 0 {
			return struct{}{}, nil
		}

		return nil, s.doBulkPrewarm(missing, requirements, onCommitProcessed)
	})
	if err != nil {
		return eris.Wrap(err, "bulk prewarm")
	}

	return nil
}

func (s *repoService) bulkPrewarmWork(
	paths map[string]bool,
	requirements metricRequirements,
) (map[string]bool, string) {
	if requirements.needsLineStats {
		missing := s.missingBulkPrewarmPaths(paths, hasLineStats)

		return missing, bulkPrewarmGroupKey(missing, "line-stats")
	}

	missing := s.missingBulkPrewarmPaths(paths, hasCommitData)

	return missing, bulkPrewarmGroupKey(missing, "metadata")
}

func hasCommitData(data *commitData) bool {
	return data != nil
}

func hasLineStats(data *commitData) bool {
	return data != nil && data.hasLineStats
}

func (s *repoService) missingBulkPrewarmPaths(
	paths map[string]bool,
	isCached func(*commitData) bool,
) map[string]bool {
	s.commitMu.RLock()
	defer s.commitMu.RUnlock()

	missing := make(map[string]bool, len(paths))
	for path := range paths {
		data := s.commitCache[path]
		if !isCached(data) {
			missing[path] = true
		}
	}

	return missing
}

func bulkPrewarmGroupKey(paths map[string]bool, requirement string) string {
	keys := make([]string, 0, len(paths))
	for path := range paths {
		keys = append(keys, path)
	}

	slices.Sort(keys)

	return requirement + ":" + strings.Join(keys, "\x00")
}

// doBulkPrewarm performs the actual bulk commit-cache population.
// It walks the entire commit history once, using tree diffs to determine
// which tracked files were modified in each commit.
func (s *repoService) doBulkPrewarm(
	paths map[string]bool,
	requirements metricRequirements,
	onCommitProcessed func(),
) error {
	cache := newBulkPrewarmCache(paths, requirements)

	if err := s.walkTrackedHistoryInRange(paths, time.Time{}, time.Time{}, onCommitProcessed, func(c *object.Commit, changed []trackedChange) {
		prewarmTrackedChanges(cache, c, changed, requirements)
	}); err != nil {
		return eris.Wrap(err, "bulk prewarm")
	}

	s.mergeBulkPrewarmCache(cache, requirements)

	return nil
}

func (s *repoService) walkTrackedHistory(
	tracked map[string]bool,
	onCommitProcessed func(),
	visit func(*object.Commit, []trackedChange),
) error {
	return s.walkTrackedHistoryInRange(tracked, time.Time{}, time.Time{}, onCommitProcessed, visit)
}

// mergeBulkPrewarmCache atomically stores results without allowing a
// metadata-only prewarm to replace complete line statistics.
func (s *repoService) mergeBulkPrewarmCache(
	cache map[string]*commitData,
	requirements metricRequirements,
) {
	isCached := hasCommitData
	if requirements.needsLineStats {
		isCached = hasLineStats
	}

	s.commitMu.Lock()
	defer s.commitMu.Unlock()

	for p, data := range cache {
		existing := s.commitCache[p]
		if isCached(existing) {
			continue
		}

		s.commitCache[p] = data
	}
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
