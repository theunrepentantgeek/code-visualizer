package git

import (
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rotisserie/eris"
)

// ContributionPoint records the contribution of one author in a single commit
// that touched a particular file.  Slices of these are retained on AuthorRecord
// so that window-based authorship metrics (initial-developer, current-maintainer,
// orphan-risk, knowledge-handoff) can filter contributions by calendar time.
type ContributionPoint struct {
	When    time.Time
	Added   int64
	Removed int64
}

// AuthorRecord holds one author's aggregate contribution to a single file.
// Contribution weight Wa = Added + Removed (removals count equally; rewards
// improving/deleting code, not just adding volume).
type AuthorRecord struct {
	Email         string
	Name          string
	Added         int64
	Removed       int64
	FirstSeen     time.Time           // earliest commit by this author touching this file
	LastSeen      time.Time           // most recent commit by this author touching this file
	Contributions []ContributionPoint // per-commit records for window-based metric calculations
}

type fileDiffStats struct {
	added   int64
	removed int64
}

// FileAuthorRecords maps repo-relative file paths (slash-separated) to
// the per-author contribution records for that file.
type FileAuthorRecords map[string][]AuthorRecord

// AuthorHistoryResult is returned by BulkAuthorHistory.
type AuthorHistoryResult struct {
	// ByFile maps repo-relative path → per-author contribution records.
	ByFile FileAuthorRecords
	// LastActive maps author email → the most recent commit date anywhere in the repo.
	// Used to compute orphan-risk and knowledge-handoff windows.
	LastActive map[string]time.Time
	// HeadDate is the repository HEAD commit date (the global clock for all
	// authorship window calculations).
	HeadDate time.Time
}

// BulkAuthorHistory walks the entire commit graph once and returns per-author
// contribution records for each tracked file, a repo-wide last-active map
// (all authors, not just those touching tracked files), and the HEAD commit date.
//
// Contribution weight per author per file: Wa = lines-added + lines-removed.
// The file-creation commit (root commit or first commit to introduce the file)
// credits lines-added to the creating author even though there is no parent diff;
// that initial line count is obtained from the blob size.
//
// For merge commits, only files that differ from ALL parents are counted
// (TREESAME simplification, matching BulkCommitHistory semantics).
//
// For non-root commits, a single tree diff is performed per commit and stats
// for all tracked changed files are extracted from that diff in one pass,
// avoiding O(N) repeated DiffTree calls for commits touching many files.
func BulkAuthorHistory(
	repoPath string,
	filePaths map[string]bool,
	honorMailmap bool,
	onCommitProcessed func(),
) (AuthorHistoryResult, error) {
	return BulkAuthorHistoryInHistoryRange(
		repoPath,
		filePaths,
		honorMailmap,
		HistoryRange{},
		onCommitProcessed,
	)
}

// BulkAuthorHistoryInRange applies the same aggregation but only considers commits
// whose author timestamps fall within the supplied date window.
//
//nolint:cyclop,funlen,maintidx,revive,nolintlint // A single-pass history walk keeps the accumulators local and coherent.
func BulkAuthorHistoryInRange(
	repoPath string,
	filePaths map[string]bool,
	honorMailmap bool,
	from time.Time,
	until time.Time,
	onCommitProcessed func(),
) (AuthorHistoryResult, error) {
	return BulkAuthorHistoryInHistoryRange(
		repoPath,
		filePaths,
		honorMailmap,
		HistoryRange{From: from, Until: until},
		onCommitProcessed,
	)
}

// BulkAuthorHistoryInHistoryRange aggregates authorship for commits selected by historyRange.
//
//nolint:cyclop,funlen,maintidx,revive,nolintlint // A single-pass history walk keeps the accumulators local and coherent.
func BulkAuthorHistoryInHistoryRange(
	repoPath string,
	filePaths map[string]bool,
	honorMailmap bool,
	historyRange HistoryRange,
	onCommitProcessed func(),
) (AuthorHistoryResult, error) {
	s, err := getService(repoPath)
	if err != nil {
		return AuthorHistoryResult{}, eris.Wrap(err, "failed to open git repository")
	}

	s.repoMu.Lock()
	defer s.repoMu.Unlock()

	commits, err := s.commitIterator(historyRange)
	if err != nil {
		return AuthorHistoryResult{}, err
	}

	var mm mailmap
	if honorMailmap {
		mm = loadMailmap(s.RepoRoot())
	}

	// per-file accumulator: path → (authorEmail → *authorAccum)
	type authorAccum struct {
		name          string
		added         int64
		removed       int64
		firstSeen     time.Time
		lastSeen      time.Time
		contributions []ContributionPoint
	}

	fileAccum := make(map[string]map[string]*authorAccum)
	lastActive := make(map[string]time.Time)
	headDate := time.Time{}

	for c, iterationErr := range commits {
		if iterationErr != nil {
			return AuthorHistoryResult{}, eris.Wrap(iterationErr, "failed to iterate commits")
		}

		when := c.Author.When
		email, name := mm.apply(c.Author.Email, c.Author.Name)

		// HEAD date is the very first commit we see (log is reverse-chronological).
		if headDate.IsZero() {
			headDate = when
		}

		// Repo-wide last-active for ALL authors.
		if prev, ok := lastActive[email]; !ok || when.After(prev) {
			lastActive[email] = when
		}

		changed := changedFilesInCommit(c, filePaths)

		if onCommitProcessed != nil {
			onCommitProcessed()
		}

		if len(changed) == 0 {
			continue
		}

		// Compute per-file line stats in one batch (single DiffTree per commit).
		var lineStats map[string]fileDiffStats
		if c.NumParents() > 0 {
			lineStats = batchFileDiffStats(c, changed)
		}

		for _, path := range changed {
			// Get or create per-file accumulator for this author.
			if fileAccum[path] == nil {
				fileAccum[path] = make(map[string]*authorAccum)
			}

			accum, exists := fileAccum[path][email]
			if !exists {
				accum = &authorAccum{name: name}
				fileAccum[path][email] = accum
			}

			// Count line contributions.
			var lineAdded, lineRemoved int64

			if c.NumParents() == 0 {
				// Root commit: credit lines in the file at this point.
				lineAdded = linesInBlob(c, path)
			} else if stats, ok := lineStats[path]; ok {
				lineAdded = stats.added
				lineRemoved = stats.removed
			}

			accum.added += lineAdded
			accum.removed += lineRemoved
			accum.contributions = append(accum.contributions, ContributionPoint{
				When:    when,
				Added:   lineAdded,
				Removed: lineRemoved,
			})

			// Update time windows.
			if accum.firstSeen.IsZero() || when.Before(accum.firstSeen) {
				accum.firstSeen = when
			}

			if accum.lastSeen.IsZero() || when.After(accum.lastSeen) {
				accum.lastSeen = when
			}
		}
	}

	// Convert accumulators to AuthorRecord slices.
	byFile := make(FileAuthorRecords, len(fileAccum))

	for path, authors := range fileAccum {
		records := make([]AuthorRecord, 0, len(authors))

		for email, accum := range authors {
			records = append(records, AuthorRecord{
				Email:         email,
				Name:          accum.name,
				Added:         accum.added,
				Removed:       accum.removed,
				FirstSeen:     accum.firstSeen,
				LastSeen:      accum.lastSeen,
				Contributions: accum.contributions,
			})
		}

		byFile[path] = records
	}

	return AuthorHistoryResult{
		ByFile:     byFile,
		LastActive: lastActive,
		HeadDate:   headDate,
	}, nil
}

// linesInBlob returns the approximate line count for a file in the given commit.
// Used to credit the creating author for the initial file contents.
// Returns 0 on any error (file not found, binary, etc.).
func linesInBlob(c *object.Commit, relPath string) int64 {
	tree, err := c.Tree()
	if err != nil {
		return 0
	}

	file, err := tree.File(relPath)
	if err != nil {
		return 0
	}

	lines, err := file.Lines()
	if err != nil {
		return 0
	}

	return int64(len(lines))
}

// batchFileDiffStats performs a single tree diff against the first parent and
// returns a map of relPath → line stats for the provided changed files.
// Using one DiffTree call per commit avoids O(N) repeated diff operations for
// commits that touch many tracked files.
//
//nolint:revive,nolintlint // The linear diff processing is intentionally kept in one function.
func batchFileDiffStats(c *object.Commit, changed []string) map[string]fileDiffStats {
	parent, err := c.Parent(0)
	if err != nil {
		return nil
	}

	parentTree, err := parent.Tree()
	if err != nil {
		return nil
	}

	commitTree, err := c.Tree()
	if err != nil {
		return nil
	}

	diffs, err := object.DiffTree(parentTree, commitTree)
	if err != nil {
		return nil
	}

	// Build a lookup set for the changed paths we care about.
	want := make(map[string]bool, len(changed))
	for _, p := range changed {
		want[p] = true
	}

	result := make(map[string]fileDiffStats, len(changed))

	for _, diff := range diffs {
		name := changeName(diff)
		if !want[name] {
			continue
		}

		patch, err := diff.Patch()
		if err != nil {
			continue
		}

		var added, removed int64

		for _, stat := range patch.Stats() {
			added += int64(stat.Addition)
			removed += int64(stat.Deletion)
		}

		result[name] = fileDiffStats{added: added, removed: removed}
	}

	return result
}
