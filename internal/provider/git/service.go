// Package git provides metric providers for git-derived metrics.
package git

import (
	"errors"
	"sync"

	gogit "github.com/go-git/go-git/v5"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rotisserie/eris"
	"golang.org/x/sync/singleflight"
)

type repoService struct {
	repo        *gogit.Repository
	rootPath    string // git worktree root (absolute path)
	commitGroup singleflight.Group
	commitMu    sync.RWMutex
	commitCache map[string]*commitData
	bulkGroup   singleflight.Group
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

	repo, err := gogit.PlainOpenWithOptions(repoPath, &gogit.PlainOpenOptions{DetectDotGit: true})
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

	svc := &repoService{
		repo:        repo,
		rootPath:    rootPath,
		commitCache: make(map[string]*commitData),
	}
	services[repoPath] = &serviceResult{svc, nil}

	return svc, nil
}

// resetService clears the cached service. Test use only.
func resetService() {
	servicesMu.Lock()
	defer servicesMu.Unlock()

	services = make(map[string]*serviceResult)
}

// RepoRootFor returns the git worktree root for the given path.
func RepoRootFor(repoPath string) (string, error) {
	s, err := getService(repoPath)
	if err != nil {
		return "", eris.Wrap(err, "failed to open git repository")
	}

	return s.RepoRoot(), nil
}

// bulkPrewarm pre-populates the commit cache for all provided file paths by
// walking the commit history once. This is dramatically faster than the default
// per-file path when many files share the same repository — e.g. 193 files
// require ~193 s with per-file git log; bulkPrewarm does it in one pass.
//
// If any paths are already cached, they are skipped. The function is safe for
// concurrent use; concurrent calls are coalesced via a singleflight group.
func (s *repoService) bulkPrewarm(paths map[string]bool) error {
	s.commitMu.RLock()

	missing := make(map[string]bool, len(paths))
	for p := range paths {
		if _, ok := s.commitCache[p]; !ok {
			missing[p] = true
		}
	}

	s.commitMu.RUnlock()

	if len(missing) == 0 {
		return nil
	}

	_, err, _ := s.bulkGroup.Do("prewarm", func() (any, error) {
		return nil, s.doBulkPrewarm(missing)
	})

	return err //nolint:wrapcheck // error already wrapped inside doBulkPrewarm
}

// doBulkPrewarm performs the actual bulk commit-cache population.
// It walks the entire commit history once, using tree diffs to determine
// which tracked files were modified in each commit.
func (s *repoService) doBulkPrewarm(paths map[string]bool) error {
	// Initialise empty commitData for all tracked paths so that untracked files
	// get a count=0 entry in the cache (avoids re-fetching them individually).
	cache := make(map[string]*commitData, len(paths))
	for p := range paths {
		cache[p] = &commitData{authors: make(map[string]bool)}
	}

	head, err := s.repo.Head()
	if err != nil {
		return eris.Wrap(err, "bulk prewarm: failed to get HEAD")
	}

	iter, err := s.repo.Log(&gogit.LogOptions{From: head.Hash()})
	if err != nil {
		return eris.Wrap(err, "bulk prewarm: failed to start git log")
	}
	defer iter.Close()

	err = iter.ForEach(s.prewarmCommit(cache, paths))
	if err != nil {
		return eris.Wrap(err, "bulk prewarm: failed to iterate commits")
	}

	// Atomically store results — only for paths not already in the cache
	// (a concurrent per-file fetch may have populated some entries first).
	s.commitMu.Lock()
	for p, data := range cache {
		if _, ok := s.commitCache[p]; !ok {
			s.commitCache[p] = data
		}
	}
	s.commitMu.Unlock()

	return nil
}

func (*repoService) prewarmCommit(
	cache map[string]*commitData,
	paths map[string]bool,
) func(c *object.Commit) error {
	return func(c *object.Commit) error {
		changed := changedFilesInCommit(c, paths)

		for _, relPath := range changed {
			data := cache[relPath]
			data.updateFrom(c, relPath)
		}

		return nil
	}
}
