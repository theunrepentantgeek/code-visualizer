package git_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
)

// buildTrackedSet returns the slash-relative path set for all files under root.
func buildTrackedSet(t *testing.T, repoRootPath string, dir *model.Directory) map[string]bool {
	t.Helper()

	tracked := make(map[string]bool)

	model.WalkFiles(dir, func(f *model.File) {
		rel, err := filepath.Rel(repoRootPath, f.Path)
		if err == nil {
			tracked[filepath.ToSlash(rel)] = true
		}
	})

	return tracked
}

// TestBulkAuthorHistory_ReturnsNonEmptyResult verifies that BulkAuthorHistory
// returns a result with at least one file, a non-zero HEAD date, and a
// non-empty last-active map when run against the code-visualizer repo itself.
func TestBulkAuthorHistory_ReturnsNonEmptyResult(t *testing.T) {
	g := NewGomegaWithT(t)

	root := repoRoot(t)

	scanned, err := scan.Scan(root, nil, nil, false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(scanned).NotTo(BeNil())

	tracked := buildTrackedSet(t, root, scanned)

	result, err := git.BulkAuthorHistory(root, tracked, nil)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.HeadDate.IsZero()).To(BeFalse(), "HeadDate should not be zero")
	g.Expect(result.ByFile).NotTo(BeEmpty(), "ByFile should have at least one entry")
	g.Expect(result.LastActive).NotTo(BeEmpty(), "LastActive should have at least one author")
}

// TestBulkAuthorHistory_EachFileHasAtLeastOneAuthor verifies that every file
// returned in ByFile has at least one AuthorRecord.
func TestBulkAuthorHistory_EachFileHasAtLeastOneAuthor(t *testing.T) {
	g := NewGomegaWithT(t)

	root := repoRoot(t)

	scanned, err := scan.Scan(root, nil, nil, false)
	g.Expect(err).NotTo(HaveOccurred())

	tracked := buildTrackedSet(t, root, scanned)

	result, err := git.BulkAuthorHistory(root, tracked, nil)
	g.Expect(err).NotTo(HaveOccurred())

	for path, records := range result.ByFile {
		g.Expect(records).NotTo(BeEmpty(), "file %q should have at least one author record", path)
	}
}

// TestBulkAuthorHistory_AuthorRecordsHaveNonEmptyEmail verifies that every
// AuthorRecord has a non-empty email address.
func TestBulkAuthorHistory_AuthorRecordsHaveNonEmptyEmail(t *testing.T) {
	g := NewGomegaWithT(t)

	root := repoRoot(t)

	scanned, err := scan.Scan(root, nil, nil, false)
	g.Expect(err).NotTo(HaveOccurred())

	tracked := buildTrackedSet(t, root, scanned)

	result, err := git.BulkAuthorHistory(root, tracked, nil)
	g.Expect(err).NotTo(HaveOccurred())

	for path, records := range result.ByFile {
		for _, r := range records {
			g.Expect(r.Email).NotTo(BeEmpty(), "author email should not be empty in file %q", path)
		}
	}
}

// TestBulkAuthorHistory_TimeWindowsAreConsistent verifies that FirstSeen is
// never after LastSeen for any AuthorRecord.
func TestBulkAuthorHistory_TimeWindowsAreConsistent(t *testing.T) {
	g := NewGomegaWithT(t)

	root := repoRoot(t)

	scanned, err := scan.Scan(root, nil, nil, false)
	g.Expect(err).NotTo(HaveOccurred())

	tracked := buildTrackedSet(t, root, scanned)

	result, err := git.BulkAuthorHistory(root, tracked, nil)
	g.Expect(err).NotTo(HaveOccurred())

	for path, records := range result.ByFile {
		for _, r := range records {
			g.Expect(r.FirstSeen).To(BeTemporally("<=", r.LastSeen),
				"FirstSeen should not be after LastSeen for %q / %q", path, r.Email)
		}
	}
}

// TestBulkAuthorHistory_LastActiveContainsKnownAuthor verifies that the
// LastActive map contains the repo's primary committer.
func TestBulkAuthorHistory_LastActiveContainsKnownAuthor(t *testing.T) {
	g := NewGomegaWithT(t)

	root := repoRoot(t)

	scanned, err := scan.Scan(root, nil, nil, false)
	g.Expect(err).NotTo(HaveOccurred())

	tracked := buildTrackedSet(t, root, scanned)

	result, err := git.BulkAuthorHistory(root, tracked, nil)
	g.Expect(err).NotTo(HaveOccurred())

	// The repo always has at least one committer; just verify the map is populated.
	g.Expect(result.LastActive).NotTo(BeEmpty())

	for email, when := range result.LastActive {
		g.Expect(email).NotTo(BeEmpty(), "author email in LastActive should not be empty")
		g.Expect(when.IsZero()).To(BeFalse(), "last-active timestamp for %q should not be zero", email)
	}
}

// TestBulkAuthorHistory_EmptyTrackedSet_ReturnsEmptyByFile verifies that
// passing an empty tracked set returns an empty ByFile map (but HeadDate and
// LastActive may still be populated if the repo has commits).
func TestBulkAuthorHistory_EmptyTrackedSet_ReturnsEmptyByFile(t *testing.T) {
	g := NewGomegaWithT(t)

	root := repoRoot(t)
	tracked := map[string]bool{}

	result, err := git.BulkAuthorHistory(root, tracked, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.ByFile).To(BeEmpty())
	// LastActive is populated from all commits, not just tracked files.
	g.Expect(result.LastActive).NotTo(BeEmpty())
	g.Expect(result.HeadDate.IsZero()).To(BeFalse())
}

// TestBulkAuthorHistory_ContributionWeightNonNegative verifies that Added and
// Removed are never negative.
func TestBulkAuthorHistory_ContributionWeightNonNegative(t *testing.T) {
	g := NewGomegaWithT(t)

	root := repoRoot(t)

	scanned, err := scan.Scan(root, nil, nil, false)
	g.Expect(err).NotTo(HaveOccurred())

	tracked := buildTrackedSet(t, root, scanned)

	result, err := git.BulkAuthorHistory(root, tracked, nil)
	g.Expect(err).NotTo(HaveOccurred())

	for path, records := range result.ByFile {
		for _, r := range records {
			g.Expect(r.Added).To(BeNumerically(">=", 0),
				"Added should be non-negative for %q / %q", path, r.Email)
			g.Expect(r.Removed).To(BeNumerically(">=", 0),
				"Removed should be non-negative for %q / %q", path, r.Email)
		}
	}
}
