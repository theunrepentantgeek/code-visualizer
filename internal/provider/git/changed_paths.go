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
	s, err := getService(repoPath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to open git repository")
	}

	changedPaths := make(map[string]bool)
	visit := func(_ *object.Commit, changes []trackedChange) {
		for _, change := range changes {
			changedPaths[change.path] = true
		}
	}

	err = s.walkTrackedHistoryInHistoryRange(
		normalizeTrackedPaths(currentPaths),
		historyRange,
		nil,
		visit,
	)
	if err != nil {
		return nil, eris.Wrap(err, "failed to find changed paths")
	}

	return changedPaths, nil
}
