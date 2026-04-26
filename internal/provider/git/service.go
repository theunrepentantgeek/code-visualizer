// Package git provides metric providers for git-derived metrics.
package git

import (
	"errors"
	"sync"
	"time"

	gogit "github.com/go-git/go-git/v5"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rotisserie/eris"
	"golang.org/x/sync/singleflight"
)

// commitData holds all per-file commit information collected in a single git log pass.
type commitData struct {
	oldest  time.Time
	newest  time.Time
	authors map[string]bool
}

type repoService struct {
	repo        *gogit.Repository
	rootPath    string // git worktree root (absolute path)
	commitGroup singleflight.Group
	commitMu    sync.RWMutex
	commitCache map[string]*commitData
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

var errUntracked = errors.New("file has no git history")

func (s *repoService) fileAge(relPath string) (int64, error) {
	data, err := s.getCommitData(relPath)
	if err != nil {
		return 0, err
	}

	if data.oldest.IsZero() {
		return 0, errUntracked
	}

	age := time.Since(data.oldest)

	return int64(age.Hours() / 24), nil
}

func (s *repoService) fileFreshness(relPath string) (int64, error) {
	data, err := s.getCommitData(relPath)
	if err != nil {
		return 0, err
	}

	if data.newest.IsZero() {
		return 0, errUntracked
	}

	freshness := time.Since(data.newest)

	return int64(freshness.Hours() / 24), nil
}

func (s *repoService) authorCount(relPath string) (int64, error) {
	data, err := s.getCommitData(relPath)
	if err != nil {
		return 0, err
	}

	if len(data.authors) == 0 {
		return 0, errUntracked
	}

	return int64(len(data.authors)), nil
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
		when := c.Author.When
		if data.oldest.IsZero() || when.Before(data.oldest) {
			data.oldest = when
		}

		if data.newest.IsZero() || when.After(data.newest) {
			data.newest = when
		}

		data.authors[c.Author.Email] = true

		return nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "failed to iterate commits")
	}

	return data, nil
}
