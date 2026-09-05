package git

import (
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
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

func (r resolvedHistoryReference) hasRevision() bool {
	return !r.revision.IsZero()
}

func (s *repoService) resolveHistoryReference(
	value string,
	bound historyBound,
) (resolvedHistoryReference, error) {
	if value == "" {
		return resolvedHistoryReference{}, nil
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
		when, err := parseHistoryDate(payload, bound)

		return resolvedHistoryReference{timestamp: when}, err
	}

	if hash, found, err := s.tryResolveTagCommit(value); found {
		return resolvedHistoryReference{revision: hash}, err
	}

	if hash, found, err := s.tryResolveCommitID(value); found {
		return resolvedHistoryReference{revision: hash}, err
	}

	if when, err := parseHistoryDate(value, bound); err == nil {
		return resolvedHistoryReference{timestamp: when}, nil
	}

	return resolvedHistoryReference{}, eris.Errorf(
		"history reference %q is not a tag, commit ID, or supported date",
		value,
	)
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

	hash, found, err := s.tryResolveCommitID(value)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	if !found {
		return plumbing.ZeroHash, eris.Errorf("unknown commit ID %q", value)
	}

	return hash, nil
}

func (s *repoService) tryResolveCommitID(value string) (plumbing.Hash, bool, error) {
	if !isCommitIDSyntax(value) {
		return plumbing.ZeroHash, false, nil
	}

	matches, err := s.hashesMatchingPrefix(strings.ToLower(value))
	if err != nil {
		return plumbing.ZeroHash, true, err
	}

	switch len(matches) {
	case 0:
		return plumbing.ZeroHash, false, nil
	case 1:
		if _, err := s.repo.CommitObject(matches[0]); err != nil {
			return plumbing.ZeroHash, true, eris.Errorf(
				"commit ID %q does not identify a commit",
				value,
			)
		}

		return matches[0], true, nil
	default:
		return plumbing.ZeroHash, true, eris.Errorf("commit ID %q is ambiguous", value)
	}
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

func parseHistoryDate(value string, bound historyBound) (time.Time, error) {
	return parseHistoryDateInLocation(value, bound, time.Now().Location())
}

func parseHistoryDateInLocation(
	value string,
	bound historyBound,
	location *time.Location,
) (time.Time, error) {
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

func historyRangeFromTimes(from, until time.Time) HistoryRange {
	var result HistoryRange

	if !from.IsZero() {
		result.From = "date:" + from.Format(time.RFC3339Nano)
	}

	if !until.IsZero() {
		result.Until = "date:" + until.Format(time.RFC3339Nano)
	}

	return result
}
