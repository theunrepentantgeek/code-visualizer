package git

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestAuthorshipMetrics(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	records := []AuthorRecord{
		{
			Email:     "early@example.test",
			Added:     6,
			FirstSeen: base,
			LastSeen:  base,
			Contributions: []ContributionPoint{
				{When: base, Added: 6},
			},
		},
		{
			Email:     "recent@example.test",
			Added:     4,
			FirstSeen: base.AddDate(0, 0, 100),
			LastSeen:  base.AddDate(0, 0, 100),
			Contributions: []ContributionPoint{
				{When: base.AddDate(0, 0, 100), Added: 4},
			},
		},
	}
	head := base.AddDate(0, 0, 100)

	g.Expect(codeOwner(records)).To(Equal("early@example.test"))
	g.Expect(initialDeveloper(records, 0.25)).To(Equal("early@example.test"))
	g.Expect(currentMaintainer(records, head, 30)).To(Equal("recent@example.test"))
	g.Expect(significantContributorCount(records, 0.1)).To(Equal(int64(2)))
	g.Expect(busFactor(records, 0.5)).To(Equal(int64(1)))
	g.Expect(ownershipDominance(records)).To(BeNumerically("~", 0.6, 0.0001))
	g.Expect(contributorEntropy(records)).To(BeNumerically("~", 0.970950594, 0.0001))
	g.Expect(orphanRisk(records, map[string]time.Time{
		"early@example.test":  base,
		"recent@example.test": head,
	}, head, 30)).To(BeNumerically("~", 0.6, 0.0001))
	g.Expect(knowledgeHandoff(records, head, 30, 0.25)).To(Equal(1.0))
}

func TestAuthorshipMetrics_SingleAuthorEntropyIsZero(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	records := []AuthorRecord{{Email: "one@example.test", Added: 1}}

	g.Expect(contributorEntropy(records)).To(Equal(0.0))
	g.Expect(busFactor(records, 0.5)).To(Equal(int64(1)))
	g.Expect(ownershipDominance(records)).To(Equal(1.0))
}

// mustTime parses an RFC3339 timestamp and panics on error.
func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}

	return t
}

// cpAt builds a ContributionPoint.
func cpAt(when string, added, removed int64) ContributionPoint {
	return ContributionPoint{When: mustTime(when), Added: added, Removed: removed}
}

// recFrom builds an AuthorRecord from ContributionPoints, computing aggregates.
func recFrom(email, name string, contribs []ContributionPoint) AuthorRecord {
	r := AuthorRecord{Email: email, Name: name, Contributions: contribs}
	for _, c := range contribs {
		r.Added += c.Added
		r.Removed += c.Removed

		if r.FirstSeen.IsZero() || c.When.Before(r.FirstSeen) {
			r.FirstSeen = c.When
		}

		if r.LastSeen.IsZero() || c.When.After(r.LastSeen) {
			r.LastSeen = c.When
		}
	}

	return r
}

func TestDefaultAuthorshipParams(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := DefaultAuthorshipParams()
	g.Expect(p.ActivityWindowDays).To(Equal(180))
	g.Expect(p.RecentWindowDays).To(Equal(180))
	g.Expect(p.EarlyWindowFraction).To(BeNumerically("~", 0.25, 1e-9))
	g.Expect(p.SignificantShareThreshold).To(BeNumerically("~", 0.10, 1e-9))
	g.Expect(p.BusFactorThreshold).To(BeNumerically("~", 0.50, 1e-9))
}

func TestSortedShares_TieBreakByFirstSeen(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	alice := recFrom("alice@x.com", "Alice", []ContributionPoint{cpAt("2023-01-01T00:00:00Z", 5, 0)})
	bob := recFrom("bob@x.com", "Bob", []ContributionPoint{cpAt("2024-01-01T00:00:00Z", 5, 0)})

	shares, _ := sortedShares([]AuthorRecord{bob, alice})
	g.Expect(shares[0].email).To(Equal("alice@x.com"), "earlier first-seen wins the tie")
}

func TestSortedShares_TieBreakByEmail(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ts := "2024-01-01T00:00:00Z"
	a := recFrom("a@x.com", "A", []ContributionPoint{cpAt(ts, 5, 0)})
	b := recFrom("b@x.com", "B", []ContributionPoint{cpAt(ts, 5, 0)})

	shares, _ := sortedShares([]AuthorRecord{b, a})
	g.Expect(shares[0].email).To(Equal("a@x.com"), "lex-earlier email wins the tie")
}

func TestCodeOwner_NoRecords_ReturnsUnmaintained(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(codeOwner(nil)).To(Equal(Unmaintained))
}

func TestInitialDeveloper_EmptyRecords_ReturnsUnmaintained(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(initialDeveloper(nil, 0.25)).To(Equal(Unmaintained))
}

func TestInitialDeveloper_PicksEarlyContributor(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Span 2020-01-01 → 2024-01-01; earlyFraction=0.25 → cutoff ~2021-01-01.
	alice := recFrom("alice@x.com", "Alice", []ContributionPoint{cpAt("2020-06-01T00:00:00Z", 100, 0)})
	bob := recFrom("bob@x.com", "Bob", []ContributionPoint{cpAt("2023-06-01T00:00:00Z", 200, 0)})

	g.Expect(initialDeveloper([]AuthorRecord{alice, bob}, 0.25)).To(Equal("alice@x.com"))
}

func TestCurrentMaintainer_NoneRecent_ReturnsUnmaintained(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	head := mustTime("2026-01-01T00:00:00Z")
	old := recFrom("a@x.com", "A", []ContributionPoint{cpAt("2022-01-01T00:00:00Z", 50, 0)})

	g.Expect(currentMaintainer([]AuthorRecord{old}, head, 180)).To(Equal(Unmaintained))
}

func TestBusFactor_NoRecords_Returns0(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(busFactor(nil, 0.50)).To(Equal(int64(0)))
}

func TestOrphanRisk_AllActive_Is0(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	head := mustTime("2024-06-01T00:00:00Z")
	alice := recFrom("alice@x.com", "Alice", []ContributionPoint{cpAt("2024-01-01T00:00:00Z", 100, 0)})
	lastActive := map[string]time.Time{"alice@x.com": mustTime("2024-05-30T00:00:00Z")}

	g.Expect(orphanRisk([]AuthorRecord{alice}, lastActive, head, 180)).To(BeNumerically("~", 0.0, 1e-9))
}

func TestOrphanRisk_MissingFromLastActive_CountsAsOrphan(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	head := mustTime("2024-06-01T00:00:00Z")
	alice := recFrom("alice@x.com", "Alice", []ContributionPoint{cpAt("2022-01-01T00:00:00Z", 100, 0)})

	g.Expect(orphanRisk([]AuthorRecord{alice}, map[string]time.Time{}, head, 180)).
		To(BeNumerically("~", 1.0, 1e-9))
}

func TestKnowledgeHandoff_YoungNode_Is0(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Node is only 10 days old; recentWindow is 365 days → overlaps → 0.
	head := mustTime("2024-01-11T00:00:00Z")
	alice := recFrom("alice@x.com", "Alice", []ContributionPoint{cpAt("2024-01-01T00:00:00Z", 100, 0)})
	bob := recFrom("bob@x.com", "Bob", []ContributionPoint{cpAt("2024-01-10T00:00:00Z", 100, 0)})

	g.Expect(knowledgeHandoff([]AuthorRecord{alice, bob}, head, 365, 0.25)).
		To(BeNumerically("~", 0.0, 1e-9))
}

func TestCollectSubtreeRecords_MergesAcrossFiles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	root := buildTree(dir, "a.go", "b.go")

	byFile := FileAuthorRecords{
		"a.go": {recFrom("alice@x.com", "Alice", []ContributionPoint{cpAt("2024-01-01T00:00:00Z", 10, 0)})},
		"b.go": {recFrom("alice@x.com", "Alice", []ContributionPoint{cpAt("2024-06-01T00:00:00Z", 20, 5)})},
	}

	records := collectSubtreeRecords(root, byFile, dir)

	g.Expect(records).To(HaveLen(1))
	g.Expect(records[0].Email).To(Equal("alice@x.com"))
	g.Expect(records[0].Added).To(Equal(int64(30)))
	g.Expect(records[0].Removed).To(Equal(int64(5)))
	g.Expect(records[0].Contributions).To(HaveLen(2))
}

func TestCollectSubtreeRecords_EmptyByFile_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	root := buildTree(dir, "a.go")

	g.Expect(collectSubtreeRecords(root, FileAuthorRecords{}, dir)).To(BeNil())
}
