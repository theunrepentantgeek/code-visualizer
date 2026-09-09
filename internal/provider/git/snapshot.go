package git

import (
	"errors"
	"io"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rotisserie/eris"
)

// Snapshot identifies the committed tree selected by an --until value.
type Snapshot struct {
	RepoRoot string
	Commit   *object.Commit
	Tree     *object.Tree
}

// ResolveSnapshot resolves until to the exact commit whose tree should be scanned.
func ResolveSnapshot(repoPath, until string) (Snapshot, error) {
	s, err := getService(repoPath)
	if err != nil {
		return Snapshot{}, eris.Wrap(err, "failed to open snapshot repository")
	}

	resolved, err := s.resolveHistoryReference(strings.TrimSpace(until), upperBound)
	if err != nil {
		return Snapshot{}, eris.Wrap(err, "invalid --until")
	}

	commit, err := s.snapshotCommit(resolved)
	if err != nil {
		return Snapshot{}, err
	}

	tree, err := commit.Tree()
	if err != nil {
		return Snapshot{}, eris.Wrapf(err, "failed to load snapshot tree for %s", commit.Hash)
	}

	return Snapshot{RepoRoot: s.RepoRoot(), Commit: commit, Tree: tree}, nil
}

func (s *repoService) snapshotCommit(resolved resolvedHistoryReference) (*object.Commit, error) {
	if resolved.hasRevision() {
		return s.snapshotRevision(resolved)
	}

	if resolved.timestamp.IsZero() {
		return nil, eris.New("--until is required to resolve a historical snapshot")
	}

	return s.snapshotDate(resolved)
}

func (s *repoService) snapshotRevision(resolved resolvedHistoryReference) (*object.Commit, error) {
	commit, err := s.repo.CommitObject(resolved.revision)
	if err != nil {
		return nil, eris.Wrapf(err, "failed to load snapshot commit %s", resolved.revision)
	}

	return commit, nil
}

func (s *repoService) snapshotDate(resolved resolvedHistoryReference) (*object.Commit, error) {
	iter, err := s.repo.Log(&gogit.LogOptions{})
	if err != nil {
		return nil, eris.Wrap(err, "failed to inspect reachable commits")
	}
	defer iter.Close()

	var selected *object.Commit

	for {
		commit, nextErr := iter.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}

		if nextErr != nil {
			return nil, eris.Wrap(nextErr, "failed to iterate reachable commits")
		}

		selected = selectSnapshotCommit(commit, selected, resolved.timestamp)
	}

	if selected == nil {
		return nil, eris.Errorf(
			"no reachable commit exists at or before %s",
			resolved.timestamp.Format("2006-01-02T15:04:05Z07:00"),
		)
	}

	return selected, nil
}

func selectSnapshotCommit(candidate, selected *object.Commit, cutoff time.Time) *object.Commit {
	if candidate.Author.When.After(cutoff) || !snapshotCommitIsLater(candidate, selected) {
		return selected
	}

	return candidate
}

func snapshotCommitIsLater(candidate, selected *object.Commit) bool {
	return selected == nil ||
		candidate.Author.When.After(selected.Author.When) ||
		(candidate.Author.When.Equal(selected.Author.When) && candidate.Hash.String() < selected.Hash.String())
}

// Subtree returns the directory tree rooted at repoRelativeDir.
func (s Snapshot) Subtree(repoRelativeDir string) (*object.Tree, error) {
	repoRelativeDir = strings.Trim(strings.ReplaceAll(repoRelativeDir, `\`, "/"), "/")
	if repoRelativeDir == "" || repoRelativeDir == "." {
		return s.Tree, nil
	}

	tree, err := s.Tree.Tree(repoRelativeDir)
	if err != nil {
		return nil, eris.Wrapf(err, "target directory %q does not exist in snapshot %s", repoRelativeDir, s.Commit.Hash)
	}

	return tree, nil
}
