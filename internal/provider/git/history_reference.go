package git

import (
	"encoding/hex"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rotisserie/eris"
)

type historyBound int

const (
	lowerBound historyBound = iota
	upperBound
)

type resolvedHistoryReference struct {
	revision  plumbing.Hash
	timestamp time.Time
}

type commitIDResolution uint8

const (
	commitIDMissing commitIDResolution = iota
	commitIDResolved
	commitIDAmbiguous
	commitIDNotCommit
)

func (r resolvedHistoryReference) hasRevision() bool {
	return !r.revision.IsZero()
}

func (s *repoService) resolveHistoryReference(
	value string,
	bound historyBound,
	timeNowValues ...time.Time,
) (resolvedHistoryReference, error) {
	if value == "" {
		return resolvedHistoryReference{}, nil
	}

	timeNow := time.Now()
	if len(timeNowValues) > 0 {
		timeNow = timeNowValues[0]
	}

	if after, ok := strings.CutPrefix(value, "^"); ok {
		return s.resolvePreviousHistoryReference(after, timeNow)
	}

	prefix, payload, explicit := splitHistoryReference(value)
	if explicit && payload == "" {
		return resolvedHistoryReference{}, eris.Errorf("%s reference cannot be empty", prefix)
	}

	switch prefix {
	case "tag":
		hash, err := s.resolveTagCommit(payload)

		return resolvedHistoryReference{revision: hash}, err
	case "sha":
		hash, err := s.requireCommitID(payload)

		return resolvedHistoryReference{revision: hash}, err
	case "date":
		when, err := parseHistoryDate(payload, bound, timeNow)

		return resolvedHistoryReference{timestamp: when}, err
	}

	return s.resolveUnprefixedHistoryReference(value, bound, timeNow)
}

func (s *repoService) resolveUnprefixedHistoryReference(
	value string,
	bound historyBound,
	timeNow time.Time,
) (resolvedHistoryReference, error) {
	if value == "HEAD" {
		head, err := s.repo.Head()
		if err != nil {
			return resolvedHistoryReference{}, eris.Wrap(err, "failed to resolve HEAD")
		}

		return resolvedHistoryReference{revision: head.Hash()}, nil
	}

	if hash, found, err := s.tryResolveTagCommit(value); found {
		return resolvedHistoryReference{revision: hash}, err
	}

	if resolved, found, err := s.tryResolveCommitID(value); found || err != nil {
		return resolved, err
	}

	if when, err := parseHistoryDate(value, bound, timeNow); err == nil {
		return resolvedHistoryReference{timestamp: when}, nil
	} else if strings.HasPrefix(value, "ago(") {
		return resolvedHistoryReference{}, err
	}

	return resolvedHistoryReference{}, eris.Errorf(
		"history reference %q is not a tag, commit ID, or supported date",
		value,
	)
}

func (s *repoService) resolvePreviousHistoryReference(
	value string,
	timeNow time.Time,
) (resolvedHistoryReference, error) {
	if value == "" {
		return resolvedHistoryReference{}, eris.New("predecessor reference cannot be empty")
	}

	if strings.HasPrefix(value, "^") {
		return resolvedHistoryReference{}, eris.New("nested predecessor references are not supported")
	}

	prefix, payload, explicit := splitHistoryReference(value)
	switch prefix {
	case "tag":
		hash, err := s.resolveTagCommit(payload)
		if err != nil {
			return resolvedHistoryReference{}, err
		}

		previous, err := s.previousTaggedCommit(hash, payload)

		return resolvedHistoryReference{revision: previous}, err
	case "sha":
		hash, err := s.requireCommitID(payload)
		if err != nil {
			return resolvedHistoryReference{}, err
		}

		previous, err := s.firstParent(hash)

		return resolvedHistoryReference{revision: previous}, err
	case "date":
		return resolvedHistoryReference{}, eris.New("predecessor requires a revision reference")
	}

	if explicit {
		return resolvedHistoryReference{}, eris.Errorf("unsupported predecessor reference %q", value)
	}

	if value == "HEAD" {
		head, err := s.repo.Head()
		if err != nil {
			return resolvedHistoryReference{}, eris.Wrap(err, "failed to resolve HEAD")
		}

		previous, err := s.firstParent(head.Hash())

		return resolvedHistoryReference{revision: previous}, err
	}

	if hash, found, err := s.tryResolveTagCommit(value); found {
		if err != nil {
			return resolvedHistoryReference{}, err
		}

		previous, previousErr := s.previousTaggedCommit(hash, value)

		return resolvedHistoryReference{revision: previous}, previousErr
	}

	if resolved, found, err := s.tryResolveCommitID(value); found || err != nil {
		if err != nil {
			return resolvedHistoryReference{}, err
		}

		previous, previousErr := s.firstParent(resolved.revision)

		return resolvedHistoryReference{revision: previous}, previousErr
	}

	if _, err := parseHistoryDate(value, lowerBound, timeNow); err == nil ||
		strings.HasPrefix(value, "ago(") {
		return resolvedHistoryReference{}, eris.New("predecessor requires a revision reference")
	}

	return resolvedHistoryReference{}, eris.Errorf(
		"history reference %q is not a tag, commit ID, or supported date",
		value,
	)
}

func (s *repoService) firstParent(hash plumbing.Hash) (plumbing.Hash, error) {
	commit, err := s.repo.CommitObject(hash)
	if err != nil {
		return plumbing.ZeroHash, eris.Wrapf(err, "failed to load commit %s", hash)
	}

	if len(commit.ParentHashes) == 0 {
		return plumbing.ZeroHash, eris.Errorf("commit %s has no first parent", hash)
	}

	return commit.ParentHashes[0], nil
}

func (s *repoService) previousTaggedCommit(
	start plumbing.Hash,
	tagName string,
) (plumbing.Hash, error) {
	tagged, err := s.taggedCommits()
	if err != nil {
		return plumbing.ZeroHash, err
	}

	startCommit, err := s.repo.CommitObject(start)
	if err != nil {
		return plumbing.ZeroHash, eris.Wrapf(err, "failed to load tag %q commit", tagName)
	}

	generation := append([]plumbing.Hash(nil), startCommit.ParentHashes...)
	seen := map[plumbing.Hash]struct{}{start: {}}

	for len(generation) > 0 {
		next := make([]plumbing.Hash, 0, len(generation))
		var selected *object.Commit

		for _, hash := range generation {
			if _, duplicate := seen[hash]; duplicate {
				continue
			}
			seen[hash] = struct{}{}

			commit, commitErr := s.repo.CommitObject(hash)
			if commitErr != nil {
				return plumbing.ZeroHash, eris.Wrapf(commitErr, "failed to traverse history from tag %q", tagName)
			}

			if _, isTagged := tagged[hash]; isTagged && laterTaggedCommit(commit, selected) {
				selected = commit
			}

			next = append(next, commit.ParentHashes...)
		}

		if selected != nil {
			return selected.Hash, nil
		}

		generation = next
	}

	return plumbing.ZeroHash, eris.Errorf("tag %q has no previous reachable tag", tagName)
}

func (s *repoService) taggedCommits() (map[plumbing.Hash]struct{}, error) {
	iter, err := s.repo.Tags()
	if err != nil {
		return nil, eris.Wrap(err, "failed to list tags")
	}
	defer iter.Close()

	tagged := make(map[plumbing.Hash]struct{})

	err = iter.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().Short()
		hash, peelErr := peelTagCommit(s.repo, name, ref.Hash())
		if peelErr == nil {
			tagged[hash] = struct{}{}
		}

		return nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "failed to iterate tags")
	}

	return tagged, nil
}

func laterTaggedCommit(candidate, selected *object.Commit) bool {
	return selected == nil ||
		candidate.Author.When.After(selected.Author.When) ||
		(candidate.Author.When.Equal(selected.Author.When) &&
			candidate.Hash.String() < selected.Hash.String())
}

func splitHistoryReference(value string) (prefix, payload string, explicit bool) {
	for _, prefix := range []string{"tag", "sha", "date"} {
		marker := prefix + ":"
		if after, ok := strings.CutPrefix(value, marker); ok {
			return prefix, after, true
		}
	}

	return "", value, false
}

func (s *repoService) tryResolveTagCommit(name string) (plumbing.Hash, bool, error) {
	ref, err := s.repo.Reference(plumbing.NewTagReferenceName(name), true)
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		return plumbing.ZeroHash, false, nil
	}

	if err != nil {
		return plumbing.ZeroHash, true, eris.Wrapf(err, "failed to resolve tag %q", name)
	}

	hash, err := peelTagCommit(s.repo, name, ref.Hash())

	return hash, true, err
}

func (s *repoService) requireCommitID(value string) (plumbing.Hash, error) {
	if !isCommitIDSyntax(value) {
		return plumbing.ZeroHash, eris.Errorf("invalid commit ID %q", value)
	}

	hash, resolution, err := s.inspectCommitID(value)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	switch resolution {
	case commitIDMissing:
		return plumbing.ZeroHash, eris.Errorf("unknown commit ID %q", value)
	case commitIDAmbiguous:
		return plumbing.ZeroHash, eris.Errorf("commit ID %q is ambiguous", value)
	case commitIDNotCommit:
		return plumbing.ZeroHash, eris.Errorf("commit ID %q does not identify a commit", value)
	case commitIDResolved:
		return hash, nil
	}

	return plumbing.ZeroHash, eris.Errorf("unknown commit ID %q", value)
}

func (s *repoService) tryResolveCommitID(
	value string,
) (resolvedHistoryReference, bool, error) {
	if !isCommitIDSyntax(value) {
		return resolvedHistoryReference{}, false, nil
	}

	hash, resolution, err := s.inspectCommitID(value)
	if err != nil {
		return resolvedHistoryReference{}, false, err
	}

	if resolution != commitIDResolved {
		return resolvedHistoryReference{}, false, nil
	}

	return resolvedHistoryReference{revision: hash}, true, nil
}

func (s *repoService) inspectCommitID(
	value string,
) (plumbing.Hash, commitIDResolution, error) {
	matches, err := s.hashesMatchingPrefix(strings.ToLower(value))
	if err != nil {
		return plumbing.ZeroHash, commitIDMissing, err
	}

	switch len(matches) {
	case 0:
		return plumbing.ZeroHash, commitIDMissing, nil
	case 1:
		if !s.isCommit(matches[0]) {
			return plumbing.ZeroHash, commitIDNotCommit, nil
		}

		return matches[0], commitIDResolved, nil
	default:
		return plumbing.ZeroHash, commitIDAmbiguous, nil
	}
}

func (s *repoService) isCommit(hash plumbing.Hash) bool {
	_, err := s.repo.CommitObject(hash)

	return err == nil
}

func (s *repoService) hashesMatchingPrefix(value string) ([]plumbing.Hash, error) {
	if len(value) == len(plumbing.ZeroHash)*2 {
		hash := plumbing.NewHash(value)
		if objectErr := s.repo.Storer.HasEncodedObject(hash); objectErr != nil {
			if errors.Is(objectErr, plumbing.ErrObjectNotFound) {
				return nil, nil
			}

			return nil, eris.Wrap(objectErr, "failed to inspect Git object")
		}

		return []plumbing.Hash{hash}, nil
	}

	evenHex := value[:len(value)&^1]

	prefix, err := hex.DecodeString(evenHex)
	if err != nil {
		return nil, eris.Wrap(err, "failed to decode commit ID")
	}

	hashes, err := s.hashesWithPrefix(prefix)
	if err != nil {
		return nil, err
	}

	if len(evenHex) == len(value) {
		return hashes, nil
	}

	return filterHashStringsByPrefix(hashes, value), nil
}

func (s *repoService) hashesWithPrefix(prefix []byte) ([]plumbing.Hash, error) {
	type prefixHasher interface {
		HashesWithPrefix(prefix []byte) ([]plumbing.Hash, error)
	}

	if indexed, ok := s.repo.Storer.(prefixHasher); ok {
		hashes, indexErr := indexed.HashesWithPrefix(prefix)

		return hashes, eris.Wrap(indexErr, "failed to inspect Git object index")
	}

	objects, err := s.repo.Storer.IterEncodedObjects(plumbing.AnyObject)
	if err != nil {
		return nil, eris.Wrap(err, "failed to inspect Git objects")
	}

	defer objects.Close()

	var hashes []plumbing.Hash

	encodedPrefix := hex.EncodeToString(prefix)

	err = objects.ForEach(func(object plumbing.EncodedObject) error {
		hash := object.Hash()
		if strings.HasPrefix(hash.String(), encodedPrefix) {
			hashes = append(hashes, hash)
		}

		return nil
	})

	return hashes, eris.Wrap(err, "failed to inspect Git objects")
}

func filterHashStringsByPrefix(hashes []plumbing.Hash, prefix string) []plumbing.Hash {
	result := make([]plumbing.Hash, 0, len(hashes))
	for _, hash := range hashes {
		if strings.HasPrefix(hash.String(), prefix) {
			result = append(result, hash)
		}
	}

	return result
}

func isCommitIDSyntax(value string) bool {
	return len(value) >= 4 && len(value) <= 40 && isHex(value)
}

func isHex(value string) bool {
	for _, char := range value {
		if (char < '0' || char > '9') &&
			(char < 'a' || char > 'f') &&
			(char < 'A' || char > 'F') {
			return false
		}
	}

	return true
}

type historyDateLayout struct {
	layout       string
	dateOnly     bool
	explicitZone bool
}

var historyDateLayouts = []historyDateLayout{
	{layout: time.RFC3339Nano, explicitZone: true},
	{layout: "2006-01-02T15:04Z07:00", explicitZone: true},
	{layout: "2006-01-02T15:04:05Z0700", explicitZone: true},
	{layout: "2006-01-02T15:04Z0700", explicitZone: true},
	{layout: "2006-01-02T15:04:05", explicitZone: false},
	{layout: "2006-01-02T15:04", explicitZone: false},
	{layout: "2006-01-02", dateOnly: true, explicitZone: false},
	{layout: "20060102Z", dateOnly: true, explicitZone: true},
	{layout: "20060102", dateOnly: true, explicitZone: false},
	{layout: "20060102-1504Z", explicitZone: true},
	{layout: "20060102-1504", explicitZone: false},
}

func parseHistoryDate(value string, bound historyBound, timeNowValues ...time.Time) (time.Time, error) {
	timeNow := time.Now()
	if len(timeNowValues) > 0 {
		timeNow = timeNowValues[0]
	}

	return parseHistoryDateInLocation(value, bound, timeNow.Location(), timeNow)
}

func parseHistoryDateInLocation(
	value string,
	bound historyBound,
	location *time.Location,
	timeNowValues ...time.Time,
) (time.Time, error) {
	timeNow := time.Now()
	if len(timeNowValues) > 0 {
		timeNow = timeNowValues[0]
	}

	if parsed, claimed, err := parseAgoDate(value, timeNow); claimed {
		return parsed, err
	}

	for _, candidate := range historyDateLayouts {
		parsed, err := parseHistoryDateLayout(value, candidate, location)
		if err != nil {
			continue
		}

		if bound == upperBound && candidate.dateOnly {
			parsed = parsed.AddDate(0, 0, 1).Add(-time.Nanosecond)
		}

		return parsed, nil
	}

	return time.Time{}, eris.Errorf(
		"invalid date %q: expected ISO 8601 or YYYYMMDD[-HHMM][Z]",
		value,
	)
}

func parseAgoDate(value string, timeNow time.Time) (time.Time, bool, error) {
	if !strings.HasPrefix(value, "ago(") {
		return time.Time{}, false, nil
	}

	if !strings.HasSuffix(value, "d)") {
		return time.Time{}, true, eris.Errorf(
			"invalid relative date %q: expected ago(<positive days>d)",
			value,
		)
	}

	daysText := strings.TrimSuffix(strings.TrimPrefix(value, "ago("), "d)")
	days, err := strconv.ParseUint(daysText, 10, 64)
	if err != nil || days == 0 {
		return time.Time{}, true, eris.Errorf(
			"invalid relative date %q: expected ago(<positive days>d)",
			value,
		)
	}

	const day = 24 * time.Hour
	if days > uint64(math.MaxInt64/int64(day)) {
		return time.Time{}, true, eris.Errorf("relative date %q exceeds supported duration", value)
	}

	return timeNow.Add(-time.Duration(days) * day), true, nil
}

func parseHistoryDateLayout(
	value string,
	candidate historyDateLayout,
	location *time.Location,
) (time.Time, error) {
	if candidate.explicitZone {
		parsed, err := time.Parse(candidate.layout, value)

		return parsed, eris.Wrap(err, "failed to parse date")
	}

	parsed, err := time.ParseInLocation(candidate.layout, value, location)

	return parsed, eris.Wrap(err, "failed to parse local date")
}
