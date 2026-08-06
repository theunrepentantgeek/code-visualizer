package git

import (
	"time"

	gogit "github.com/go-git/go-git/v5"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rotisserie/eris"
)

// AuthorRecord holds one author's aggregate contribution to a single file.
// Contribution weight Wa = Added + Removed (removals count equally; rewards
// improving/deleting code, not just adding volume).
type AuthorRecord struct {
	Email     string
	Name      string
	Added     int64
	Removed   int64
	FirstSeen time.Time // earliest commit by this author touching this file
	LastSeen  time.Time // most recent commit by this author touching this file
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
//
//nolint:revive // A single-pass history walk keeps the accumulators local and coherent.
func BulkAuthorHistory(
	repoPath string,
	filePaths map[string]bool,
	onCommitProcessed func(),
) (AuthorHistoryResult, error) {
	s, err := getService(repoPath)
	if err != nil {
		return AuthorHistoryResult{}, eris.Wrap(err, "failed to open git repository")
	}

	head, err := s.repo.Head()
	if err != nil {
		return AuthorHistoryResult{}, eris.Wrap(err, "failed to get HEAD")
	}

	iter, err := s.repo.Log(&gogit.LogOptions{From: head.Hash()})
	if err != nil {
		return AuthorHistoryResult{}, eris.Wrap(err, "failed to start log iteration")
	}
	defer iter.Close()

	// per-file accumulator: path → (authorEmail → *authorAccum)
	type authorAccum struct {
		name      string
		added     int64
		removed   int64
		firstSeen time.Time
		lastSeen  time.Time
	}
	fileAccum := make(map[string]map[string]*authorAccum)
	lastActive := make(map[string]time.Time)
	headDate := time.Time{}

	err = iter.ForEach(func(c *object.Commit) error {
		when := c.Author.When
		email := c.Author.Email
		name := c.Author.Name

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
			return nil
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
			if c.NumParents() == 0 {
				// Root commit: credit lines in the file at this point.
				accum.added += linesInBlob(c, path)
			} else if stats, ok := lineStats[path]; ok {
				accum.added += stats.added
				accum.removed += stats.removed
			}

			// Update time windows.
			if accum.firstSeen.IsZero() || when.Before(accum.firstSeen) {
				accum.firstSeen = when
			}

			if accum.lastSeen.IsZero() || when.After(accum.lastSeen) {
				accum.lastSeen = when
			}
		}

		return nil
	})
	if err != nil {
		return AuthorHistoryResult{}, eris.Wrap(err, "failed to iterate commits")
	}

	// Convert accumulators to AuthorRecord slices.
	byFile := make(FileAuthorRecords, len(fileAccum))

	for path, authors := range fileAccum {
		records := make([]AuthorRecord, 0, len(authors))

		for email, accum := range authors {
			records = append(records, AuthorRecord{
				Email:     email,
				Name:      accum.name,
				Added:     accum.added,
				Removed:   accum.removed,
				FirstSeen: accum.firstSeen,
				LastSeen:  accum.lastSeen,
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
//nolint:revive // The linear diff processing is intentionally kept in one function.
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
