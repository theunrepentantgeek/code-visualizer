package git

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestResolveSnapshotSelectsRevisionAndLatestDateCommit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fixture := setupTagRangeRepo(t)

	tagged, err := ResolveSnapshot(fixture.dir, "tag:v1.0")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(tagged.Commit.Hash.String()).To(Equal(fixture.initial))

	dated, err := ResolveSnapshot(fixture.dir, "date:2025-03-01")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(dated.Commit.Hash.String()).To(Equal(fixture.feature))
	g.Expect(dated.Commit.Author.When.Equal(time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC))).To(BeTrue())
}

func TestResolveSnapshotReportsNoCommitBeforeDate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fixture := setupTagRangeRepo(t)

	_, err := ResolveSnapshot(fixture.dir, "date:2024-01-01")
	g.Expect(err).To(MatchError(ContainSubstring("no reachable commit")))
}
