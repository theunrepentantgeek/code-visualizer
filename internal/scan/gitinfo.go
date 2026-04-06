package scan

import (
	"bytes"
	"errors"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rotisserie/eris"
)

// IsGitRepo checks if the given path is inside a git repository.
func IsGitRepo(path string) (bool, error) {
	_, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return false, nil
		}

		return false, eris.Wrap(err, "failed to check git repository")
	}

	return true, nil
}

// GitInfo provides git metadata extraction for files in a repository.
type GitInfo struct {
	repo     *git.Repository
	headTree *object.Tree
}

// NewGitInfo opens the git repository at the given path.
func NewGitInfo(path string) (*GitInfo, error) {
	repo, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, eris.Wrap(err, "failed to open git repository")
	}

	return &GitInfo{repo: repo}, nil
}

// ClearCache discards the cached HEAD tree so subsequent calls
// re-read from the repository. Call this after a bulk operation completes.
func (g *GitInfo) ClearCache() {
	g.headTree = nil
}

func (g *GitInfo) getHeadTree() (*object.Tree, error) {
	if g.headTree != nil {
		return g.headTree, nil
	}

	if g.repo == nil {
		return nil, eris.New("git repository not initialized")
	}

	head, err := g.repo.Head()
	if err != nil {
		return nil, eris.Wrap(err, "failed to get HEAD reference")
	}

	commit, err := g.repo.CommitObject(head.Hash())
	if err != nil {
		return nil, eris.Wrap(err, "failed to get HEAD commit")
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, eris.Wrap(err, "failed to get commit tree")
	}

	g.headTree = tree

	return tree, nil
}

// ErrUntracked indicates a file has no git history (untracked).
var ErrUntracked = errors.New("file has no git history")

// FileAge returns the duration since the file's first commit.
// Returns ErrUntracked if the file has no git history.
func (g *GitInfo) FileAge(relPath string) (*time.Duration, error) {
	commits, err := g.fileCommits(relPath)
	if err != nil {
		return nil, err
	}

	if len(commits) == 0 {
		return nil, ErrUntracked
	}
	// First commit is the oldest (last in the chronological log)
	oldest := commits[len(commits)-1]
	age := time.Since(oldest)

	return &age, nil
}

// FileFreshness returns the duration since the file's most recent commit.
// Returns ErrUntracked if the file has no git history.
func (g *GitInfo) FileFreshness(relPath string) (*time.Duration, error) {
	commits, err := g.fileCommits(relPath)
	if err != nil {
		return nil, err
	}

	if len(commits) == 0 {
		return nil, ErrUntracked
	}
	// Most recent commit is the first in the log
	newest := commits[0]
	freshness := time.Since(newest)

	return &freshness, nil
}

// AuthorCount returns the number of distinct committers for the file.
// Returns ErrUntracked if the file has no git history.
func (g *GitInfo) AuthorCount(relPath string) (*int, error) {
	if g.repo == nil {
		return nil, eris.New("git repository not initialized")
	}

	log, err := g.repo.Log(&git.LogOptions{
		FileName: &relPath,
	})
	if err != nil {
		return nil, eris.Wrap(err, "failed to get git log")
	}
	defer log.Close()

	authors := map[string]bool{}

	err = log.ForEach(func(c *object.Commit) error {
		authors[c.Author.Email] = true

		return nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "failed to iterate commits")
	}

	if len(authors) == 0 {
		return nil, ErrUntracked
	}

	count := len(authors)

	return &count, nil
}

// IsBinary checks if a file is binary by inspecting its content in the HEAD tree.
func (g *GitInfo) IsBinary(relPath string) bool {
	tree, err := g.getHeadTree()
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
	defer reader.Close()

	// Read first 8000 bytes to check for null bytes (same heuristic as git)
	buf := make([]byte, 8000)
	n, _ := reader.Read(buf)

	return bytes.Contains(buf[:n], []byte{0})
}

// fileCommits returns commit times for a file, newest first.
func (g *GitInfo) fileCommits(relPath string) ([]time.Time, error) {
	if g.repo == nil {
		return nil, eris.New("git repository not initialized")
	}

	log, err := g.repo.Log(&git.LogOptions{
		FileName: &relPath,
	})
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
