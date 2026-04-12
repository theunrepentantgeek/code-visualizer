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

var errUntracked = errors.New("file has no git history")

func (s *repoService) fileAge(relPath string) (int, error) {
	commits, err := s.fileCommitTimes(relPath)
	if err != nil {
		return 0, err
	}

	if len(commits) == 0 {
		return 0, errUntracked
	}

	oldest := commits[len(commits)-1]
	age := time.Since(oldest)

	return int(age.Seconds()), nil
}

func (s *repoService) fileFreshness(relPath string) (int, error) {
	commits, err := s.fileCommitTimes(relPath)
	if err != nil {
		return 0, err
	}

	if len(commits) == 0 {
		return 0, errUntracked
	}

	newest := commits[0]
	freshness := time.Since(newest)

	return int(freshness.Seconds()), nil
}

func (s *repoService) authorCount(relPath string) (int, error) {
	log, err := s.repo.Log(&gogit.LogOptions{FileName: &relPath})
	if err != nil {
		return 0, eris.Wrap(err, "failed to get git log")
	}
	defer log.Close()

	authors := map[string]bool{}

	err = log.ForEach(func(c *object.Commit) error {
		authors[c.Author.Email] = true

		return nil
	})
	if err != nil {
		return 0, eris.Wrap(err, "failed to iterate commits")
	}

	if len(authors) == 0 {
		return 0, errUntracked
	}

	return len(authors), nil
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
