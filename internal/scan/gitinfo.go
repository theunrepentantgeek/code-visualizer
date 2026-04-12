package scan

import (
	"errors"

	"github.com/go-git/go-git/v5"
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
