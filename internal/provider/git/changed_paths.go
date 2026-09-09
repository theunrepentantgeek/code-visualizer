package git

import (
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rotisserie/eris"
)

// ChangedPathsInHistoryRange returns currentPaths modified by at least one
// commit selected by historyRange.
func ChangedPathsInHistoryRange(
	repoPath string,
	currentPaths map[string]bool,
	historyRange HistoryRange,
) (map[string]bool, error) {
	return changedPathsInHistoryRange(repoPath, currentPaths, historyRange, true)
}

// ChangedPathsInHistoryRangeForSnapshot treats currentPaths as the authoritative
// path set from a historical tree rather than intersecting the live index.
func ChangedPathsInHistoryRangeForSnapshot(
	repoPath string,
	currentPaths map[string]bool,
	historyRange HistoryRange,
) (map[string]bool, error) {
	return changedPathsInHistoryRange(repoPath, currentPaths, historyRange, false)
}

func changedPathsInHistoryRange(
	repoPath string,
	currentPaths map[string]bool,
	historyRange HistoryRange,
	intersectIndex bool,
) (map[string]bool, error) {
	s, err := getService(repoPath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to open git repository")
	}

	trackedPaths := normalizeTrackedPaths(currentPaths)
	if intersectIndex {
		trackedPaths, err = s.intersectIndexPaths(currentPaths)
		if err != nil {
			return nil, eris.Wrap(err, "failed to find current tracked paths")
		}
	}

	changedPaths := make(map[string]bool)
	visit := func(_ *object.Commit, changes []trackedChange) {
		for _, change := range changes {
			changedPaths[change.path] = true
		}
	}

	err = s.walkTrackedHistoryInHistoryRange(
		trackedPaths,
		historyRange,
		nil,
		visit,
	)
	if err != nil {
		return nil, eris.Wrap(err, "failed to find changed paths")
	}

	return changedPaths, nil
}

func (s *repoService) intersectIndexPaths(currentPaths map[string]bool) (map[string]bool, error) {
	s.repoMu.Lock()
	defer s.repoMu.Unlock()

	index, err := s.repo.Storer.Index()
	if err != nil {
		return nil, eris.Wrap(err, "failed to read git index")
	}

	currentPaths = normalizeTrackedPaths(currentPaths)
	trackedPaths := make(map[string]bool, len(currentPaths))

	for _, entry := range index.Entries {
		if currentPaths[entry.Name] {
			trackedPaths[entry.Name] = true
		}
	}

	return trackedPaths, nil
}
