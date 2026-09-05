package git

import (
	"errors"
	"io"
	"iter"
	"strconv"
	"time"

	gogit "github.com/go-git/go-git/v5"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rotisserie/eris"
)

// HistoryRange selects commits by graph position and author timestamp.
type HistoryRange struct {
	From     time.Time
	Until    time.Time
	FromTag  string
	UntilTag string
}

type resolvedHistoryRange struct {
	tip      plumbing.Hash
	excluded map[plumbing.Hash]struct{}
	from     time.Time
	until    time.Time
}

func (s *repoService) resolveHistoryRange(r HistoryRange) (resolvedHistoryRange, error) {
	tip, tipLabel, err := s.resolveRangeTip(r.UntilTag)
	if err != nil {
		return resolvedHistoryRange{}, err
	}

	resolved := resolvedHistoryRange{
		tip:   tip,
		from:  r.From,
		until: r.Until,
	}
	if r.FromTag == "" {
		return resolved, nil
	}

	from, err := s.resolveTagCommit(r.FromTag)
	if err != nil {
		return resolvedHistoryRange{}, err
	}

	reachableFromTip, err := s.reachableHashes(tip)
	if err != nil {
		return resolvedHistoryRange{}, eris.Wrap(err, "failed to inspect effective tip history")
	}

	if _, ok := reachableFromTip[from]; !ok {
		return resolvedHistoryRange{}, eris.Errorf(
			"tag %q is not an ancestor of %s",
			r.FromTag,
			tipLabel,
		)
	}

	excluded, err := s.reachableHashes(from)
	if err != nil {
		return resolvedHistoryRange{}, eris.Wrapf(err, "failed to inspect tag %q history", r.FromTag)
	}

	resolved.excluded = excluded

	return resolved, nil
}

func (s *repoService) resolveRangeTip(untilTag string) (plumbing.Hash, string, error) {
	if untilTag != "" {
		hash, err := s.resolveTagCommit(untilTag)

		return hash, "tag " + strconv.Quote(untilTag), err
	}

	head, err := s.repo.Head()
	if err != nil {
		return plumbing.ZeroHash, "", eris.Wrap(err, "failed to get HEAD")
	}

	return head.Hash(), "HEAD", nil
}

func (s *repoService) resolveTagCommit(name string) (plumbing.Hash, error) {
	ref, err := s.repo.Reference(plumbing.NewTagReferenceName(name), true)
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return plumbing.ZeroHash, eris.Errorf("tag %q not found", name)
		}

		return plumbing.ZeroHash, eris.Wrapf(err, "failed to resolve tag %q", name)
	}

	hash := ref.Hash()
	seen := make(map[plumbing.Hash]struct{})

	for {
		if _, err := s.repo.CommitObject(hash); err == nil {
			return hash, nil
		}

		if _, duplicate := seen[hash]; duplicate {
			return plumbing.ZeroHash, eris.Errorf("tag %q contains a tag cycle", name)
		}

		seen[hash] = struct{}{}

		tag, err := s.repo.TagObject(hash)
		if err != nil {
			return plumbing.ZeroHash, eris.Errorf("tag %q does not reference a commit", name)
		}

		hash = tag.Target
	}
}

func (s *repoService) reachableHashes(from plumbing.Hash) (map[plumbing.Hash]struct{}, error) {
	commitIter, err := s.repo.Log(&gogit.LogOptions{From: from})
	if err != nil {
		return nil, eris.Wrap(err, "failed to start reachable-history iteration")
	}

	defer commitIter.Close()

	result := make(map[plumbing.Hash]struct{})

	err = commitIter.ForEach(func(commit *object.Commit) error {
		result[commit.Hash] = struct{}{}

		return nil
	})

	return result, eris.Wrap(err, "failed to iterate reachable history")
}

func (s *repoService) commitIterator(r HistoryRange) (iter.Seq2[*object.Commit, error], error) {
	resolved, err := s.resolveHistoryRange(r)
	if err != nil {
		return nil, err
	}

	commitIter, err := s.repo.Log(&gogit.LogOptions{From: resolved.tip})
	if err != nil {
		return nil, eris.Wrap(err, "failed to start log iteration")
	}

	return filterCommitsInRange(commitSequence(commitIter), resolved), nil
}

func commitSequence(commitIter object.CommitIter) iter.Seq2[*object.Commit, error] {
	return func(yield func(*object.Commit, error) bool) {
		yieldCommits(commitIter, yield)
	}
}

func yieldCommits(commitIter object.CommitIter, yield func(*object.Commit, error) bool) {
	defer commitIter.Close()

	for {
		commit, done, iterationErr := nextCommit(commitIter)
		if done {
			return
		}

		if iterationErr != nil {
			yield(nil, iterationErr)

			return
		}

		if !yield(commit, nil) {
			return
		}
	}
}

func nextCommit(commitIter object.CommitIter) (*object.Commit, bool, error) {
	commit, err := commitIter.Next()
	if errors.Is(err, io.EOF) {
		return nil, true, nil
	}

	return commit, false, eris.Wrap(err, "failed to read commit")
}

func filterCommitsInRange(
	commits iter.Seq2[*object.Commit, error],
	r resolvedHistoryRange,
) iter.Seq2[*object.Commit, error] {
	return func(yield func(*object.Commit, error) bool) {
		for commit, iterationErr := range commits {
			if !yieldCommitInRange(yield, commit, iterationErr, r) {
				return
			}
		}
	}
}

func yieldCommitInRange(
	yield func(*object.Commit, error) bool,
	commit *object.Commit,
	iterationErr error,
	historyRange resolvedHistoryRange,
) bool {
	if iterationErr != nil {
		yield(nil, iterationErr)

		return false
	}

	if _, excluded := historyRange.excluded[commit.Hash]; excluded {
		return true
	}

	return !commitInDateRange(commit, historyRange.from, historyRange.until) || yield(commit, nil)
}

func commitInDateRange(commit *object.Commit, from time.Time, until time.Time) bool {
	return (from.IsZero() || !commit.Author.When.Before(from)) &&
		(until.IsZero() || !commit.Author.When.After(until))
}
