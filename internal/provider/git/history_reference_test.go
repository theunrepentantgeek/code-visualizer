package git

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage"
)

type fixedPrefixStorer struct {
	storage.Storer
	hashes []plumbing.Hash
}

func (s *fixedPrefixStorer) HashesWithPrefix([]byte) ([]plumbing.Hash, error) {
	return s.hashes, nil
}

func TestResolveHistoryReference_UsesExplicitPrefixes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	s := requireRepoService(t, fixture.dir)

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

	s := requireRepoService(t, fixture.dir)

	resolved, err := s.resolveHistoryReference("sha:"+fixture.main, lowerBound)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.revision).To(Equal(plumbing.NewHash(fixture.main)))
}

func TestResolveHistoryReference_CommitPredecessorsUseFirstParent(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)
	s := requireRepoService(t, fixture.dir)

	for _, value := range []string{
		"^HEAD",
		"^" + fixture.merged[:8],
		"^sha:" + fixture.merged[:8],
	} {
		resolved, err := s.resolveHistoryReference(value, lowerBound)
		g.Expect(err).NotTo(HaveOccurred(), value)
		g.Expect(resolved.revision).To(Equal(plumbing.NewHash(fixture.main)), value)
	}
}

func TestResolveHistoryReference_TagPredecessorUsesNearestTaggedGeneration(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)
	s := requireRepoService(t, fixture.dir)

	for _, value := range []string{"^v2.0", "^tag:v2.0", "^latest-lightweight"} {
		resolved, err := s.resolveHistoryReference(value, lowerBound)
		g.Expect(err).NotTo(HaveOccurred(), value)
		g.Expect(resolved.revision).To(Equal(plumbing.NewHash(fixture.featureTip)), value)
	}
}

func TestResolveHistoryReference_TagPredecessorStopsAtFirstTaggedGeneration(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)
	runHistoryGit(t, fixture.dir, "tag", "deeper-but-later", fixture.feature)
	s := requireRepoService(t, fixture.dir)

	resolved, err := s.resolveHistoryReference("^v2.0", lowerBound)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.revision).To(Equal(plumbing.NewHash(fixture.featureTip)))
}

func TestResolveHistoryReference_RejectsInvalidPredecessors(t *testing.T) {
	t.Parallel()
	fixture := setupTagRangeRepo(t)
	s := requireRepoService(t, fixture.dir)

	tests := []struct {
		value   string
		message string
	}{
		{"^", "predecessor reference cannot be empty"},
		{"^^HEAD", "nested predecessor references are not supported"},
		{"^" + fixture.initial, "has no first parent"},
		{"^v1.0", `tag "v1.0" has no previous reachable tag`},
		{"^date:2025-01-01", "predecessor requires a revision reference"},
		{"^ago(7d)", "predecessor requires a revision reference"},
		{"^sha:deadbeef", `unknown commit ID "deadbeef"`},
		{"^tag:blob-tag", `tag "blob-tag" does not reference a commit`},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)
			_, err := s.resolveHistoryReference(tt.value, lowerBound)
			g.Expect(err).To(MatchError(ContainSubstring(tt.message)))
		})
	}
}

func TestResolveHistoryReference_UnprefixedTagWinsOverCommitAndDate(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)
	runHistoryGit(t, fixture.dir, "tag", fixture.main[:8], fixture.initial)
	runHistoryGit(t, fixture.dir, "tag", "20250501", fixture.main)

	s := requireRepoService(t, fixture.dir)

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

	s := requireRepoService(t, fixture.dir)

	resolved, err := s.resolveHistoryReference(fixture.main[:8], lowerBound)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.revision).To(Equal(plumbing.NewHash(fixture.main)))
}

func TestResolveHistoryReference_CompactDateFallsThroughCommitIDCollisions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		hashes []plumbing.Hash
	}{
		{
			name:   "non-commit object",
			hashes: []plumbing.Hash{plumbing.NewHash("1111111111111111111111111111111111111111")},
		},
		{
			name: "ambiguous prefix",
			hashes: []plumbing.Hash{
				plumbing.NewHash("1111111111111111111111111111111111111111"),
				plumbing.NewHash("2222222222222222222222222222222222222222"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)
			fixture := setupTagRangeRepo(t)

			s := requireRepoService(t, fixture.dir)
			s.repo.Storer = &fixedPrefixStorer{
				Storer: s.repo.Storer,
				hashes: tt.hashes,
			}

			resolved, err := s.resolveHistoryReference("20260905", lowerBound)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(resolved.timestamp.Format("20060102-150405")).To(Equal("20260905-000000"))
		})
	}
}

func TestResolveHistoryReference_RejectsInvalidExplicitReferences(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	s := requireRepoService(t, fixture.dir)

	_, err := s.resolveHistoryReference("tag:", lowerBound)
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

	s := requireRepoService(t, fixture.dir)

	prefix := addAmbiguousObjectPrefix(t, s)
	_, err := s.resolveHistoryReference("sha:"+prefix, lowerBound)
	g.Expect(err).To(MatchError(ContainSubstring(`commit ID "` + prefix + `" is ambiguous`)))
}

func TestResolveHistoryReference_RejectsNonCommitObjectID(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)
	blob := runHistoryGit(t, fixture.dir, "rev-parse", "blob-tag^{blob}")

	s := requireRepoService(t, fixture.dir)

	_, err := s.resolveHistoryReference("sha:"+blob, lowerBound)
	g.Expect(err).To(MatchError(ContainSubstring("does not identify a commit")))
}

func TestResolveHistoryReference_ReportsUnknownValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)

	s := requireRepoService(t, fixture.dir)

	_, err := s.resolveHistoryReference("not-a-reference", lowerBound)
	g.Expect(err).To(MatchError(
		`history reference "not-a-reference" is not a tag, commit ID, or supported date`,
	))
}

func TestParseAgoDate_UsesCapturedTimeNow(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	timeNow := time.Date(2026, 10, 3, 3, 20, 57, 451_000_000, time.FixedZone("NZDT", 13*60*60))

	got, claimed, err := parseAgoDate("ago(7d)", timeNow)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(claimed).To(BeTrue())
	g.Expect(got).To(Equal(timeNow.Add(-7 * 24 * time.Hour)))
}

func TestResolveHistoryReference_AcceptsRelativeDates(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)
	s := requireRepoService(t, fixture.dir)
	timeNow := time.Date(2026, 10, 3, 3, 20, 57, 0, time.UTC)

	unprefixed, err := s.resolveHistoryReference("ago(7d)", lowerBound, timeNow)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(unprefixed.timestamp).To(Equal(timeNow.Add(-7 * 24 * time.Hour)))

	explicit, err := s.resolveHistoryReference("date:ago(14d)", upperBound, timeNow)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(explicit.timestamp).To(Equal(timeNow.Add(-14 * 24 * time.Hour)))
}

func TestParseAgoDate_RejectsInvalidExpressions(t *testing.T) {
	t.Parallel()

	tests := []string{
		"ago(0d)",
		"ago(-1d)",
		"ago(1.5d)",
		"ago(7)",
		"ago(7h)",
		"ago(999999999999999999999999999d)",
		"ago()",
		"ago(7d",
	}

	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			_, claimed, err := parseAgoDate(value, time.Now())

			g.Expect(claimed).To(BeTrue())
			g.Expect(err).To(HaveOccurred())
		})
	}
}

func TestResolveHistoryReference_RejectsRelativeDatePredecessor(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)
	s := requireRepoService(t, fixture.dir)

	_, err := s.resolveHistoryReference("^ago(7d)", lowerBound, time.Now())

	g.Expect(err).To(MatchError(ContainSubstring("predecessor requires a revision reference")))
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

func requireRepoService(t *testing.T, dir string) *repoService {
	t.Helper()
	g := NewGomegaWithT(t)

	s, err := getService(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("expected git repository service")
	}

	return s
}

func runHistoryGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	NewWithT(t).Expect(err).NotTo(HaveOccurred(), string(output))

	return strings.TrimSpace(string(output))
}

func addAmbiguousObjectPrefix(t *testing.T, s *repoService) string {
	t.Helper()
	g := NewGomegaWithT(t)
	seen := make(map[string]plumbing.Hash)
	repo := s.repo

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
