package git

import (
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
		if strings.HasPrefix(value, marker) {
			return prefix, strings.TrimPrefix(value, marker), true
		}
	}

	return "", value, false
}

func (s *repoService) tryResolveTagCommit(name string) (plumbing.Hash, bool, error) {
	_, err := s.repo.Reference(plumbing.NewTagReferenceName(name), true)
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		return plumbing.ZeroHash, false, nil
	}

	if err != nil {
		return plumbing.ZeroHash, true, eris.Wrapf(err, "failed to resolve tag %q", name)
	}

	hash, err := s.resolveTagCommit(name)

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

	objects, err := s.repo.Storer.IterEncodedObjects(plumbing.AnyObject)
	if err != nil {
		return plumbing.ZeroHash, true, eris.Wrap(err, "failed to inspect Git objects")
	}

	defer objects.Close()

	var matches []plumbing.EncodedObject

	err = objects.ForEach(func(object plumbing.EncodedObject) error {
		if strings.HasPrefix(object.Hash().String(), strings.ToLower(value)) {
			matches = append(matches, object)
		}

		return nil
	})
	if err != nil {
		return plumbing.ZeroHash, true, eris.Wrap(err, "failed to inspect Git objects")
	}

	switch len(matches) {
	case 0:
		return plumbing.ZeroHash, false, nil
	case 1:
		if matches[0].Type() != plumbing.CommitObject {
			return plumbing.ZeroHash, true, eris.Errorf(
				"commit ID %q does not identify a commit",
				value,
			)
		}

		return matches[0].Hash(), true, nil
	default:
		return plumbing.ZeroHash, true, eris.Errorf("commit ID %q is ambiguous", value)
	}
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
	{layout: "2006-01-02T15:04:05", explicitZone: false},
	{layout: "2006-01-02T15:04", explicitZone: false},
	{layout: "2006-01-02", dateOnly: true, explicitZone: false},
	{layout: "20060102Z", dateOnly: true, explicitZone: true},
	{layout: "20060102", dateOnly: true, explicitZone: false},
	{layout: "20060102-1504Z", explicitZone: true},
	{layout: "20060102-1504", explicitZone: false},
}

func parseHistoryDate(value string, bound historyBound) (time.Time, error) {
	for _, candidate := range historyDateLayouts {
		parsed, err := parseHistoryDateLayout(value, candidate)
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

func parseHistoryDateLayout(value string, candidate historyDateLayout) (time.Time, error) {
	if candidate.explicitZone {
		return time.Parse(candidate.layout, value)
	}

	return time.ParseInLocation(candidate.layout, value, time.Local)
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
