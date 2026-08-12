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
