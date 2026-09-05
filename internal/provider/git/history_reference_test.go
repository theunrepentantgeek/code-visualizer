package git

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/go-git/go-git/v5/plumbing"
)

func TestResolveHistoryReference_UsesExplicitPrefixes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())

	tag, err := s.resolveHistoryReference("tag:v1.0", lowerBound)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(tag.revision).To(Equal(plumbing.NewHash(fixture.initial)))

	sha, err := s.resolveHistoryReference("sha:"+fixture.main[:8], lowerBound)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(sha.revision).To(Equal(plumbing.NewHash(fixture.main)))

	date, err := s.resolveHistoryReference("date:20250905-1430Z", lowerBound)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(date.timestamp).To(Equal(time.Date(2025, 9, 5, 14, 30, 0, 0, time.UTC)))
}

func TestResolveHistoryReference_AcceptsFullCommitID(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())

	resolved, err := s.resolveHistoryReference("sha:"+fixture.main, lowerBound)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.revision).To(Equal(plumbing.NewHash(fixture.main)))
}

func TestResolveHistoryReference_UnprefixedTagWinsOverCommitAndDate(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)
	runHistoryGit(t, fixture.dir, "tag", fixture.main[:8], fixture.initial)
	runHistoryGit(t, fixture.dir, "tag", "20250501", fixture.main)

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())

	shortCollision, err := s.resolveHistoryReference(fixture.main[:8], lowerBound)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(shortCollision.revision).To(Equal(plumbing.NewHash(fixture.initial)))

	dateCollision, err := s.resolveHistoryReference("20250501", lowerBound)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(dateCollision.revision).To(Equal(plumbing.NewHash(fixture.main)))
}

func TestResolveHistoryReference_UnprefixedCommitWinsOverDate(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())

	resolved, err := s.resolveHistoryReference(fixture.main[:8], lowerBound)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.revision).To(Equal(plumbing.NewHash(fixture.main)))
}

func TestResolveHistoryReference_RejectsInvalidExplicitReferences(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())

	_, err = s.resolveHistoryReference("tag:", lowerBound)
	g.Expect(err).To(MatchError(ContainSubstring("tag reference cannot be empty")))

	_, err = s.resolveHistoryReference("sha:", lowerBound)
	g.Expect(err).To(MatchError(ContainSubstring("sha reference cannot be empty")))

	_, err = s.resolveHistoryReference("date:", lowerBound)
	g.Expect(err).To(MatchError(ContainSubstring("date reference cannot be empty")))

	_, err = s.resolveHistoryReference("sha:not-a-hash", lowerBound)
	g.Expect(err).To(MatchError(ContainSubstring(`invalid commit ID "not-a-hash"`)))

	_, err = s.resolveHistoryReference("sha:deadbeef", lowerBound)
	g.Expect(err).To(MatchError(ContainSubstring(`unknown commit ID "deadbeef"`)))

	_, err = s.resolveHistoryReference("date:tomorrow", lowerBound)
	g.Expect(err).To(MatchError(ContainSubstring(`invalid date "tomorrow"`)))
}

func TestResolveHistoryReference_RejectsAmbiguousCommitID(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())

	prefix := addAmbiguousObjectPrefix(t, s)
	_, err = s.resolveHistoryReference("sha:"+prefix, lowerBound)
	g.Expect(err).To(MatchError(ContainSubstring(`commit ID "` + prefix + `" is ambiguous`)))
}

func TestResolveHistoryReference_RejectsNonCommitObjectID(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)
	blob := runHistoryGit(t, fixture.dir, "rev-parse", "blob-tag^{blob}")

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())

	_, err = s.resolveHistoryReference("sha:"+blob, lowerBound)
	g.Expect(err).To(MatchError(ContainSubstring("does not identify a commit")))
}

func TestResolveHistoryReference_ReportsUnknownValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())

	_, err = s.resolveHistoryReference("not-a-reference", lowerBound)
	g.Expect(err).To(MatchError(
		`history reference "not-a-reference" is not a tag, commit ID, or supported date`,
	))
}

func TestParseHistoryDate_SupportsDocumentedFormats(t *testing.T) {
	t.Parallel()

	local := time.FixedZone("test-local", 12*60*60)

	tests := []struct {
		value string
		want  time.Time
	}{
		{"2026-09-05", time.Date(2026, 9, 5, 0, 0, 0, 0, local)},
		{"20260905", time.Date(2026, 9, 5, 0, 0, 0, 0, local)},
		{"20260905Z", time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)},
		{"20260905-1430", time.Date(2026, 9, 5, 14, 30, 0, 0, local)},
		{"20260905-1430Z", time.Date(2026, 9, 5, 14, 30, 0, 0, time.UTC)},
		{"2026-09-05T14:30:45Z", time.Date(2026, 9, 5, 14, 30, 45, 0, time.UTC)},
		{"2026-09-05T14:30:45+12:00", time.Date(2026, 9, 5, 14, 30, 45, 0, local)},
		{"2026-09-05T14:30:45+1200", time.Date(2026, 9, 5, 14, 30, 45, 0, local)},
		{"2026-09-05T14:30+12:00", time.Date(2026, 9, 5, 14, 30, 0, 0, local)},
		{"2026-09-05T14:30+1200", time.Date(2026, 9, 5, 14, 30, 0, 0, local)},
		{"2026-09-05T14:30:45", time.Date(2026, 9, 5, 14, 30, 45, 0, local)},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)
			got, err := parseHistoryDateInLocation(tt.value, lowerBound, local)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(got.Equal(tt.want)).To(BeTrue())
		})
	}
}

func TestParseHistoryDate_UpperDateIncludesEntireDay(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	local := time.FixedZone("test-local", 12*60*60)

	got, err := parseHistoryDateInLocation("20260905", upperBound, local)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got).To(Equal(
		time.Date(2026, 9, 6, 0, 0, 0, 0, local).Add(-time.Nanosecond),
	))
}

func TestParseHistoryDate_UpperTimestampUsesExactInstant(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	got, err := parseHistoryDate("2026-09-05T14:30:45Z", upperBound)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got).To(Equal(time.Date(2026, 9, 5, 14, 30, 45, 0, time.UTC)))
}

func runHistoryGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...) //nolint:gosec // fixed test command
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	NewWithT(t).Expect(err).NotTo(HaveOccurred(), string(output))

	return strings.TrimSpace(string(output))
}

func addAmbiguousObjectPrefix(t *testing.T, s *repoService) string {
	t.Helper()
	g := NewGomegaWithT(t)
	seen := make(map[string]plumbing.Hash)
	repo, err := s.repository()
	g.Expect(err).NotTo(HaveOccurred())

	if repo == nil {
		t.Fatal("expected Git repository")
	}

	for i := range 10_000 {
		encoded := repo.Storer.NewEncodedObject()
		encoded.SetType(plumbing.BlobObject)

		writer, err := encoded.Writer()
		g.Expect(err).NotTo(HaveOccurred())

		if writer == nil {
			t.Fatal("expected Git object writer")
		}

		_, err = fmt.Fprintf(writer, "collision candidate %d", i)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(writer.Close()).To(Succeed())

		hash, err := repo.Storer.SetEncodedObject(encoded)
		g.Expect(err).NotTo(HaveOccurred())

		prefix := hash.String()[:4]
		if previous, ok := seen[prefix]; ok && previous != hash {
			return prefix
		}

		seen[prefix] = hash
	}

	t.Fatal("failed to generate a four-character object ID collision")

	return ""
}
