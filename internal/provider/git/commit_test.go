package git

import (
	"path/filepath"
	"testing"
	"time"

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

func TestCommitTotal_ReturnsReachableCommitCount(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	total, err := CommitTotal(setupTestGitRepo(t))

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(total).To(Equal(int64(3)))
}

func TestCommitTotalInRange_ReturnsOnlyCommitsInWindow(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)

	from := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC)

	total, err := CommitTotalInRange(dir, from, until)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(total).To(Equal(int64(1)))
}

//nolint:paralleltest // resetService mutates the global service registry used by cache assertions.
func TestHistoryRange_TotalHistoryAndPrewarmUseSameSelection(t *testing.T) {
	g := NewGomegaWithT(t)
	fixture := setupTagRangeRepo(t)
	historyRange := HistoryRange{FromTag: "v1.0", UntilTag: "v2.0"}
	tracked := map[string]bool{"main.go": true, "feature.go": true}

	total, err := CommitTotalInHistoryRange(fixture.dir, historyRange)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(total).To(Equal(int64(4)))

	resetService()

	processed := 0
	commits, err := BulkCommitHistoryAndPrewarmInHistoryRange(
		fixture.dir,
		tracked,
		[]metric.Name{CommitCount},
		historyRange,
		func() { processed++ },
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(processed).To(Equal(int(total)))
	g.Expect(commits).To(HaveLen(3))

	s, err := getService(fixture.dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(s).NotTo(BeNil())

	if s == nil {
		t.Fatal("expected git repository service")
	}

	mainData := s.cachedCommitData("main.go")
	featureData := s.cachedCommitData("feature.go")

	g.Expect(mainData).NotTo(BeNil())
	g.Expect(featureData).NotTo(BeNil())

	if mainData == nil || featureData == nil {
		t.Fatal("expected prewarmed commit data")
	}

	g.Expect(mainData.count).To(Equal(int64(1)))
	g.Expect(featureData.count).To(Equal(int64(2)))
}

func TestCommitIterator_SupportsRangeIteration(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s, err := getService(setupTestGitRepo(t))
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("expected git repository service")
	}

	from := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC)

	commits, err := s.commitIterator(HistoryRange{From: from, Until: until})
	g.Expect(err).NotTo(HaveOccurred())

	var count int

	for _, iterationErr := range commits {
		g.Expect(iterationErr).NotTo(HaveOccurred())

		count++
	}

	g.Expect(count).To(Equal(1))
}

//nolint:paralleltest // resetService mutates the global service registry used by cache assertions.
func TestBulkCommitHistoryAndPrewarm_PreservesHistoryAndWarmsCache(t *testing.T) {
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	tracked := map[string]bool{"shared.go": true}

	resetService()

	historyOnly, err := BulkCommitHistory(dir, tracked, nil)
	g.Expect(err).NotTo(HaveOccurred())

	resetService()

	historyAndPrewarm, err := BulkCommitHistoryAndPrewarm(dir, tracked, []metric.Name{CommitCount}, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(historyAndPrewarm).To(Equal(historyOnly))

	s, err := getService(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("expected git repository service")
	}

	g.Expect(s.cachedCommitData("shared.go")).NotTo(BeNil())
}

//nolint:paralleltest // resetService mutates the global service registry used by cache assertions.
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

	if s == nil {
		t.Fatal("expected git repository service")
	}

	cached := s.cachedCommitData("shared.go")
	if cached == nil {
		t.Fatal("expected prewarmed commit data")
	}

	g.Expect(cached.count).To(Equal(int64(2)))
	g.Expect(cached.hasLineStats).To(BeFalse())
}

//nolint:paralleltest // resetService mutates the global service registry used by cache assertions.
func TestBulkCommitHistoryAndPrewarm_WarmsEquivalentChurnData(t *testing.T) {
	g := NewGomegaWithT(t)

	dir := setupDiffRepo(t)

	resetService()

	_, err := BulkCommitHistoryAndPrewarm(
		dir,
		map[string]bool{"churn.go": true},
		[]metric.Name{TotalLinesAdded, TotalLinesRemoved},
		nil,
	)
	g.Expect(err).NotTo(HaveOccurred())

	s, err := getService(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("expected git repository service")
	}

	cached := s.cachedCommitData("churn.go")
	if cached == nil {
		t.Fatal("expected prewarmed commit data")
	}

	expected, err := s.fetchCommitData("churn.go")
	g.Expect(err).NotTo(HaveOccurred())

	if expected == nil {
		t.Fatal("expected direct commit data")
	}

	g.Expect(cached.hasLineStats).To(BeTrue())
	g.Expect(cached.linesAdded).To(Equal(expected.linesAdded))
	g.Expect(cached.linesRemoved).To(Equal(expected.linesRemoved))
	g.Expect(cached.linesAdded).To(Equal(int64(3)))
	g.Expect(cached.linesRemoved).To(Equal(int64(1)))
}

func TestNormalizeTrackedPaths_UsesNativeSeparators(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	normalized := normalizeTrackedPaths(map[string]bool{
		"subdir" + string(filepath.Separator) + "code.go": true,
	})

	g.Expect(normalized).To(HaveKey("subdir/code.go"))
}

func TestNormalizeTrackedPaths_PreservesBackslashesInPOSIXFilenames(t *testing.T) {
	t.Parallel()

	if filepath.Separator == '\\' {
		t.Skip("backslash is a path separator on Windows")
	}

	g := NewGomegaWithT(t)
	path := `legal\filename.go`

	normalized := normalizeTrackedPaths(map[string]bool{path: true})

	g.Expect(normalized).To(HaveKey(path))
	g.Expect(normalized).NotTo(HaveKey("legal/filename.go"))
}

//nolint:paralleltest // resetService mutates the global service registry used by cache assertions.
func TestLoadGitMetrics_ReusesCombinedPrewarmCache(t *testing.T) {
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	tracked := map[string]bool{"shared.go": true}

	resetService()

	_, err := BulkCommitHistoryAndPrewarm(dir, tracked, []metric.Name{CommitCount}, nil)
	g.Expect(err).NotTo(HaveOccurred())

	s, err := getService(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("expected git repository service")
	}

	cached := s.cachedCommitData("shared.go")
	if cached == nil {
		t.Fatal("expected prewarmed commit data")
	}

	root := buildTree(dir, "shared.go")
	g.Expect(loadGitMetrics(root, []metric.Name{CommitCount}, nil)).To(Succeed())
	g.Expect(s.cachedCommitData("shared.go")).To(BeIdenticalTo(cached))

	count, ok := root.Files[0].Quantity(CommitCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(count).To(Equal(int64(2)))
}

//nolint:paralleltest // resetService mutates the global service registry used by cache assertions.
func TestLoadGitMetrics_ReusesCombinedPrewarmCacheForSubdirectoryTarget(t *testing.T) {
	g := NewGomegaWithT(t)

	subdir := setupSubdirRepo(t)
	trackedPath := filepath.ToSlash(filepath.Join(filepath.Base(subdir), "code.go"))

	resetService()

	repoRoot, err := RepoRootFor(subdir)
	g.Expect(err).NotTo(HaveOccurred())

	s, err := getService(subdir)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("expected git repository service")
	}

	_, err = BulkCommitHistoryAndPrewarm(
		repoRoot,
		map[string]bool{trackedPath: true},
		[]metric.Name{CommitCount},
		nil,
	)
	g.Expect(err).NotTo(HaveOccurred())

	cached := s.cachedCommitData(trackedPath)
	if cached == nil {
		t.Fatal("expected prewarmed commit data")
	}

	root := buildTree(subdir, "code.go")
	g.Expect(loadGitMetrics(root, []metric.Name{CommitCount}, nil)).To(Succeed())
	g.Expect(s.cachedCommitData(trackedPath)).To(BeIdenticalTo(cached))

	count, ok := root.Files[0].Quantity(CommitCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(count).To(Equal(int64(1)))
}
