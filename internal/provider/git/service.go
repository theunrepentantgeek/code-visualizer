// Package git provides metric providers for git-derived metrics.
package git

import (
	"errors"
	"sync"
	"time"

	gogit "github.com/go-git/go-git/v5"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rotisserie/eris"
)

type repoService struct {
	repo *gogit.Repository
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

	svc := &repoService{repo: repo}
	services[repoPath] = &serviceResult{svc, nil}

	return svc, nil
}

// resetService clears the cached service. Test use only.
func resetService() {
	servicesMu.Lock()
	defer servicesMu.Unlock()

	services = make(map[string]*serviceResult)
}

// VerifyRepository returns an error if repoPath is not within a git repository.
func VerifyRepository(repoPath string) error {
	_, err := getService(repoPath)

	return err
}

// FileAuthors returns the set of distinct author emails for the file at relPath.
// relPath must be relative to the repository root. repoPath is searched upward for a .git directory.
// Returns an empty map for untracked files (no error).
func FileAuthors(repoPath, relPath string) (map[string]bool, error) {
	s, err := getService(repoPath)
	if err != nil {
		return nil, err
	}

	return s.fileAuthors(relPath)
}

var errUntracked = errors.New("file has no git history")

func (s *repoService) fileAge(relPath string) (int64, error) {
	commits, err := s.fileCommitTimes(relPath)
	if err != nil {
		return 0, err
	}

	if len(commits) == 0 {
		return 0, errUntracked
	}

	oldest := commits[len(commits)-1]
	age := time.Since(oldest)

	return int64(age.Hours() / 24), nil
}

func (s *repoService) fileFreshness(relPath string) (int64, error) {
	commits, err := s.fileCommitTimes(relPath)
	if err != nil {
		return 0, err
	}

	if len(commits) == 0 {
		return 0, errUntracked
	}

	newest := commits[0]
	freshness := time.Since(newest)

	return int64(freshness.Hours() / 24), nil
}

func (s *repoService) authorCount(relPath string) (int64, error) {
	authors, err := s.fileAuthors(relPath)
	if err != nil {
		return 0, err
	}

	if len(authors) == 0 {
		return 0, errUntracked
	}

	return int64(len(authors)), nil
}

func (s *repoService) fileAuthors(relPath string) (map[string]bool, error) {
	log, err := s.repo.Log(&gogit.LogOptions{FileName: &relPath})
	if err != nil {
		return nil, eris.Wrap(err, "failed to get git log")
	}
	defer log.Close()

	authors := make(map[string]bool)

	err = log.ForEach(func(c *object.Commit) error {
		authors[c.Author.Email] = true

		return nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "failed to iterate commits")
	}

	return authors, nil
}

func (s *repoService) fileCommitTimes(relPath string) ([]time.Time, error) {
	log, err := s.repo.Log(&gogit.LogOptions{FileName: &relPath})
	if err != nil {
		return nil, eris.Wrap(err, "failed to get git log")
	}
	defer log.Close()

	var times []time.Time

	err = log.ForEach(func(c *object.Commit) error {
		times = append(times, c.Author.When)

		return nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "failed to iterate commits")
	}

	return times, nil
}
