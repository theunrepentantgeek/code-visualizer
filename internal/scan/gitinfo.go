package scan

import (
	"bytes"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// IsGitRepo checks if the given path is inside a git repository.
func IsGitRepo(path string) (bool, error) {
	_, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		if err == git.ErrRepositoryNotExists {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GitInfo provides git metadata extraction for files in a repository.
type GitInfo struct {
	repo *git.Repository
}

// NewGitInfo opens the git repository at the given path.
func NewGitInfo(path string) (*GitInfo, error) {
	repo, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, err
	}
	return &GitInfo{repo: repo}, nil
}

// FileAge returns the duration since the file's first commit.
// Returns nil if the file has no git history (untracked).
func (g *GitInfo) FileAge(relPath string) (*time.Duration, error) {
	commits, err := g.fileCommits(relPath)
	if err != nil {
		return nil, err
	}
	if len(commits) == 0 {
		return nil, nil
	}
	// First commit is the oldest (last in the chronological log)
	oldest := commits[len(commits)-1]
	age := time.Since(oldest)
	return &age, nil
}

// FileFreshness returns the duration since the file's most recent commit.
// Returns nil if the file has no git history (untracked).
func (g *GitInfo) FileFreshness(relPath string) (*time.Duration, error) {
	commits, err := g.fileCommits(relPath)
	if err != nil {
		return nil, err
	}
	if len(commits) == 0 {
		return nil, nil
	}
	// Most recent commit is the first in the log
	newest := commits[0]
	freshness := time.Since(newest)
	return &freshness, nil
}

// AuthorCount returns the number of distinct committers for the file.
// Returns nil if the file has no git history (untracked).
func (g *GitInfo) AuthorCount(relPath string) (*int, error) {
	log, err := g.repo.Log(&git.LogOptions{
		FileName: &relPath,
	})
	if err != nil {
		return nil, err
	}
	defer log.Close()

	authors := map[string]bool{}
	err = log.ForEach(func(c *object.Commit) error {
		authors[c.Author.Email] = true
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(authors) == 0 {
		return nil, nil
	}
	count := len(authors)
	return &count, nil
}

// IsBinary checks if a file is binary by inspecting its content in the HEAD tree.
func (g *GitInfo) IsBinary(relPath string) bool {
	head, err := g.repo.Head()
	if err != nil {
		return false
	}

	commit, err := g.repo.CommitObject(head.Hash())
	if err != nil {
		return false
	}

	tree, err := commit.Tree()
	if err != nil {
		return false
	}

	file, err := tree.File(relPath)
	if err != nil {
		return false
	}

	reader, err := file.Reader()
	if err != nil {
		return false
	}
	defer reader.Close() //nolint:errcheck

	// Read first 8000 bytes to check for null bytes (same heuristic as git)
	buf := make([]byte, 8000)
	n, _ := reader.Read(buf)
	return bytes.Contains(buf[:n], []byte{0})
}

// fileCommits returns commit times for a file, newest first.
func (g *GitInfo) fileCommits(relPath string) ([]time.Time, error) {
	log, err := g.repo.Log(&git.LogOptions{
		FileName: &relPath,
	})
	if err != nil {
		return nil, err
	}
	defer log.Close()

	var times []time.Time
	err = log.ForEach(func(c *object.Commit) error {
		times = append(times, c.Author.When)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return times, nil
}
