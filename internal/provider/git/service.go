// Package git provides metric providers for git-derived metrics.
package git

import (
	"errors"
	"sync"

	gogit "github.com/go-git/go-git/v5"

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
