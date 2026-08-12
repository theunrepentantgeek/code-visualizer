package git

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// ─── globalEmailRanking ────────────────────────────────────────────────────────

func TestGlobalEmailRanking_TopKFilters(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	byFile := FileAuthorRecords{
		"a.go": {
			recFrom("alice@x.com", "Alice", []ContributionPoint{cpAt("2024-01-01T00:00:00Z", 80, 0)}),
			recFrom("bob@x.com", "Bob", []ContributionPoint{cpAt("2024-01-01T00:00:00Z", 15, 0)}),
			recFrom("carol@x.com", "Carol", []ContributionPoint{cpAt("2024-01-01T00:00:00Z", 5, 0)}),
		},
	}

	top := globalEmailRanking(byFile, 2)
	g.Expect(top).To(HaveKey("alice@x.com"))
	g.Expect(top).To(HaveKey("bob@x.com"))
	g.Expect(top).NotTo(HaveKey("carol@x.com"))
}

func TestGlobalEmailRanking_SumsAcrossFiles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	byFile := FileAuthorRecords{
		"a.go": {recFrom("alice@x.com", "Alice", []ContributionPoint{cpAt("2024-01-01T00:00:00Z", 10, 0)})},
		"b.go": {recFrom("alice@x.com", "Alice", []ContributionPoint{cpAt("2024-01-01T00:00:00Z", 90, 0)})},
	}

	top := globalEmailRanking(byFile, 1)
	g.Expect(top).To(HaveKey("alice@x.com"))
}

// ─── bucketIdentityMetrics ─────────────────────────────────────────────────────

func buildModelTree(dir string, files ...string) *model.Directory {
	root := &model.Directory{Path: dir, Name: filepath.Base(dir)}
	for _, name := range files {
		root.Files = append(root.Files, &model.File{
			Path: filepath.Join(dir, name),
			Name: name,
		})
	}

	return root
}

func TestBucketIdentityMetrics_BucketsSmallContributors(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	root := buildModelTree(dir, "a.go")

	// Set classification on the file directly.
	root.Files[0].SetClassification(CodeOwnerMetric, "carol@x.com")

	byFile := FileAuthorRecords{
		"a.go": {
			recFrom("alice@x.com", "Alice", []ContributionPoint{cpAt("2024-01-01T00:00:00Z", 80, 0)}),
			recFrom("bob@x.com", "Bob", []ContributionPoint{cpAt("2024-01-01T00:00:00Z", 15, 0)}),
			recFrom("carol@x.com", "Carol", []ContributionPoint{cpAt("2024-01-01T00:00:00Z", 5, 0)}),
		},
	}

	// With topK=2, carol is beyond top-2 and should become OtherContributor.
	bucketIdentityMetrics(root, byFile, 2)

	val, ok := root.Files[0].Classification(CodeOwnerMetric)
	g.Expect(ok).To(BeTrue())
	g.Expect(val).To(Equal(OtherContributor))
}

func TestBucketIdentityMetrics_PreservesTopK(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	root := buildModelTree(dir, "a.go")

	root.Files[0].SetClassification(CodeOwnerMetric, "alice@x.com")

	byFile := FileAuthorRecords{
		"a.go": {
			recFrom("alice@x.com", "Alice", []ContributionPoint{cpAt("2024-01-01T00:00:00Z", 80, 0)}),
			recFrom("bob@x.com", "Bob", []ContributionPoint{cpAt("2024-01-01T00:00:00Z", 5, 0)}),
		},
	}

	bucketIdentityMetrics(root, byFile, 1)

	val, ok := root.Files[0].Classification(CodeOwnerMetric)
	g.Expect(ok).To(BeTrue())
	g.Expect(val).To(Equal("alice@x.com"))
}

func TestBucketIdentityMetrics_PreservesUnmaintained(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	root := buildModelTree(dir, "a.go")

	root.Files[0].SetClassification(CurrentMaintainerMetric, Unmaintained)

	byFile := FileAuthorRecords{
		"a.go": {
			recFrom("alice@x.com", "Alice", []ContributionPoint{cpAt("2024-01-01T00:00:00Z", 100, 0)}),
		},
	}

	bucketIdentityMetrics(root, byFile, 0) // topK=0 → everyone beyond top-0

	val, ok := root.Files[0].Classification(CurrentMaintainerMetric)
	g.Expect(ok).To(BeTrue())
	// Unmaintained is always preserved regardless of topK.
	g.Expect(val).To(Equal(Unmaintained))
}
