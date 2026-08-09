package git

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

func TestBulkCommitHistory_ReturnsCommitsForTrackedFiles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	tracked := map[string]bool{
		"old.go":    true,
		"shared.go": true,
		"new.go":    true,
	}

	commits, err := BulkCommitHistory(dir, tracked, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(commits).NotTo(BeEmpty())

	for _, c := range commits {
		g.Expect(c.Hash).NotTo(BeEmpty())
		g.Expect(c.Author.Name).NotTo(BeEmpty())
		g.Expect(c.Author.When.IsZero()).To(BeFalse())
		g.Expect(c.ChangedPaths).NotTo(BeEmpty())
	}
}

func TestBulkCommitHistory_CapturesAuthorIdentity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	tracked := map[string]bool{"shared.go": true}

	commits, err := BulkCommitHistory(dir, tracked, nil)
	g.Expect(err).NotTo(HaveOccurred())

	authors := map[string]bool{}
	for _, c := range commits {
		authors[c.Author.Name] = true
	}

	g.Expect(authors).To(HaveKey("Alice"))
	g.Expect(authors).To(HaveKey("Bob"))
}

func TestBulkCommitHistory_SkipsCommitsNotTouchingTracked(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	tracked := map[string]bool{"old.go": true}

	commits, err := BulkCommitHistory(dir, tracked, nil)
	g.Expect(err).NotTo(HaveOccurred())

	for _, c := range commits {
		g.Expect(c.ChangedPaths).To(ContainElement("old.go"))
	}
}

func TestBulkCommitHistory_InvokesProgressCallback(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	tracked := map[string]bool{"old.go": true, "shared.go": true, "new.go": true}

	count := 0

	_, err := BulkCommitHistory(dir, tracked, func() { count++ })
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(count).To(BeNumerically(">=", 1))
}

func TestBulkCommitHistoryAndPrewarm_ReturnsHistoryAndWarmsCommitCount(t *testing.T) {
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	tracked := map[string]bool{"shared.go": true}

	resetService()

	commits, err := BulkCommitHistoryAndPrewarm(dir, tracked, []metric.Name{CommitCount}, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(commits).To(HaveLen(2))

	s, err := getService(dir)
	g.Expect(err).NotTo(HaveOccurred())

	cached := s.cachedCommitData("shared.go")
	g.Expect(cached).NotTo(BeNil())
	g.Expect(cached.count).To(Equal(int64(2)))
	g.Expect(cached.hasLineStats).To(BeFalse())
}

func TestBulkCommitHistoryAndPrewarm_NormalizesTrackedPaths(t *testing.T) {
	g := NewGomegaWithT(t)

	dir := setupSubdirRepo(t)
	trackedPath := filepath.ToSlash(filepath.Join(filepath.Base(dir), "code.go"))

	resetService()

	commits, err := BulkCommitHistoryAndPrewarm(
		dir,
		map[string]bool{"subdir\\code.go": true},
		[]metric.Name{CommitCount},
		nil,
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(commits).To(HaveLen(1))
	g.Expect(commits[0].ChangedPaths).To(ConsistOf(trackedPath))

	s, err := getService(dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(s.cachedCommitData(trackedPath)).NotTo(BeNil())
}

func TestLoadGitMetrics_ReusesCombinedPrewarmCache(t *testing.T) {
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	tracked := map[string]bool{"shared.go": true}

	resetService()

	_, err := BulkCommitHistoryAndPrewarm(dir, tracked, []metric.Name{CommitCount}, nil)
	g.Expect(err).NotTo(HaveOccurred())

	s, err := getService(dir)
	g.Expect(err).NotTo(HaveOccurred())

	cached := s.cachedCommitData("shared.go")
	g.Expect(cached).NotTo(BeNil())

	root := buildTree(dir, "shared.go")
	g.Expect(loadGitMetrics(root, []metric.Name{CommitCount}, nil)).To(Succeed())
	g.Expect(s.cachedCommitData("shared.go")).To(BeIdenticalTo(cached))

	count, ok := root.Files[0].Quantity(CommitCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(count).To(Equal(int64(2)))
}

func TestLoadGitMetrics_ReusesCombinedPrewarmCacheForSubdirectoryTarget(t *testing.T) {
	g := NewGomegaWithT(t)

	subdir := setupSubdirRepo(t)
	trackedPath := filepath.ToSlash(filepath.Join(filepath.Base(subdir), "code.go"))

	resetService()

	repoRoot, err := RepoRootFor(subdir)
	g.Expect(err).NotTo(HaveOccurred())

	s, err := getService(subdir)
	g.Expect(err).NotTo(HaveOccurred())

	_, err = BulkCommitHistoryAndPrewarm(
		repoRoot,
		map[string]bool{trackedPath: true},
		[]metric.Name{CommitCount},
		nil,
	)
	g.Expect(err).NotTo(HaveOccurred())

	cached := s.cachedCommitData(trackedPath)
	g.Expect(cached).NotTo(BeNil())

	root := buildTree(subdir, "code.go")
	g.Expect(loadGitMetrics(root, []metric.Name{CommitCount}, nil)).To(Succeed())
	g.Expect(s.cachedCommitData(trackedPath)).To(BeIdenticalTo(cached))

	count, ok := root.Files[0].Quantity(CommitCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(count).To(Equal(int64(1)))
}
