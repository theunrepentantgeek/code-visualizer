package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

func setupTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()

		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper
		cmd.Dir = dir

		cmd.Env = append(
			os.Environ(),
			"GIT_AUTHOR_NAME=Alice",
			"GIT_AUTHOR_EMAIL=alice@example.com",
			"GIT_COMMITTER_NAME=Alice",
			"GIT_COMMITTER_EMAIL=alice@example.com",
		)

		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %s\n%s", args, err, out)
		}
	}

	runAs := func(name, email string, args ...string) {
		t.Helper()

		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper
		cmd.Dir = dir

		cmd.Env = append(
			os.Environ(),
			"GIT_AUTHOR_NAME="+name,
			"GIT_AUTHOR_EMAIL="+email,
			"GIT_COMMITTER_NAME="+name,
			"GIT_COMMITTER_EMAIL="+email,
		)

		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %s\n%s", args, err, out)
		}
	}

	run("git", "init")
	run("git", "config", "user.name", "Alice")
	run("git", "config", "user.email", "alice@example.com")

	_ = os.WriteFile(filepath.Join(dir, "old.go"), []byte("package main\n"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "shared.go"), []byte("package shared\n"), 0o600)

	run("git", "add", ".")
	run("git", "commit", "-m", "initial commit", "--date=2024-01-01T00:00:00+00:00")

	_ = os.WriteFile(filepath.Join(dir, "shared.go"), []byte("package shared\n// updated by bob\n"), 0o600)

	runAs("Bob", "bob@example.com", "git", "add", "shared.go")
	runAs("Bob", "bob@example.com", "git", "commit", "-m", "bob update", "--date=2025-06-15T00:00:00+00:00")

	_ = os.WriteFile(filepath.Join(dir, "new.go"), []byte("package new\n"), 0o600)

	run("git", "add", "new.go")
	run("git", "commit", "-m", "add new.go")

	return dir
}

func buildTree(dir string, files ...string) *model.Directory {
	root := &model.Directory{Path: dir, Name: filepath.Base(dir)}

	for _, name := range files {
		root.Files = append(root.Files, &model.File{
			Path: filepath.Join(dir, name),
			Name: name,
		})
	}

	return root
}

func TestRepoRelativePath_UsesSlashSeparators(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	repoRoot := filepath.Join("repo", "root")
	path := filepath.Join(repoRoot, "nested", "file.go")

	rel, err := repoRelativePath(repoRoot, path)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(rel).To(Equal("nested/file.go"))
}

func TestIsGitMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(IsGitMetric(FileAge)).To(BeTrue())
	g.Expect(IsGitMetric(FileFreshness)).To(BeTrue())
	g.Expect(IsGitMetric(AuthorCount)).To(BeTrue())
	g.Expect(IsGitMetric(CommitCount)).To(BeTrue())
	g.Expect(IsGitMetric("file-size")).To(BeFalse())
	g.Expect(IsGitMetric("file-lines")).To(BeFalse())
	g.Expect(IsGitMetric("unknown-metric")).To(BeFalse())
}

func TestFileAgeProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	root := buildTree(dir, "old.go", "new.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(Succeed())

	// old.go has age > 0
	ageOld, ok := root.Files[0].Quantity(FileAge)
	g.Expect(ok).To(BeTrue())
	g.Expect(ageOld).To(BeNumerically(">", 0))

	// new.go has age >= 0 (just committed)
	ageNew, ok := root.Files[1].Quantity(FileAge)
	g.Expect(ok).To(BeTrue())
	g.Expect(ageNew).To(BeNumerically(">=", 0))

	// old.go should be older than new.go
	g.Expect(ageOld).To(BeNumerically(">", ageNew))
}

func TestMetricsLoaderReportsFileProgressThroughoutPrewarm(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	root := buildTree(dir, "old.go", "new.go")

	var processed atomic.Int64
	firstProgressBeforeMetrics := false

	resetService()

	loader := &metricsLoader{
		onFile: func() {
			if processed.Add(1) == 1 {
				_, metricApplied := root.Files[0].Quantity(CommitCount)
				firstProgressBeforeMetrics = !metricApplied
			}
		},
	}

	g.Expect(loader.Load(root, []metric.Name{CommitCount})).To(Succeed())
	g.Expect(processed.Load()).To(Equal(int64(2)))
	g.Expect(firstProgressBeforeMetrics).To(BeTrue())
}

func TestFileProgressCallbacksScaleCommitsAcrossFiles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var processed int64
	callbacks := newFileProgressCallbacks(func() {
		processed++
	}, 10, 4)

	callbacks.onPrewarm()
	g.Expect(processed).To(Equal(int64(2)))
	callbacks.onPrewarm()
	g.Expect(processed).To(Equal(int64(4)))
	callbacks.onPrewarm()
	g.Expect(processed).To(Equal(int64(6)))
	callbacks.onPrewarm()
	g.Expect(processed).To(Equal(int64(9)))
	callbacks.onPrewarm()
	g.Expect(processed).To(Equal(int64(9)))

	callbacks.onFinish()
	g.Expect(processed).To(Equal(int64(10)))
}

//nolint:paralleltest // resetService mutates the global service registry used by cache assertions.
func TestLoadGitMetrics_ReportsProgressFromCombinedPrewarmCache(t *testing.T) {
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	root := buildTree(dir, "old.go", "new.go")

	resetService()

	_, err := BulkCommitHistoryAndPrewarm(
		dir,
		map[string]bool{"old.go": true, "new.go": true},
		[]metric.Name{CommitCount},
		nil,
	)
	g.Expect(err).NotTo(HaveOccurred())

	var processed atomic.Int64

	g.Expect(loadGitMetrics(root, []metric.Name{CommitCount}, func() {
		processed.Add(1)
	})).To(Succeed())
	g.Expect(processed.Load()).To(Equal(int64(2)))
}

func TestMetricsLoaderLoadsOnlyRequestedCommitCount(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	root := buildTree(dir, "shared.go")

	resetService()
	g.Expect((&metricsLoader{}).Load(root, []metric.Name{CommitCount})).To(Succeed())

	count, countOK := root.Files[0].Quantity(CommitCount)
	g.Expect(countOK).To(BeTrue())
	g.Expect(count).To(Equal(int64(2)))

	_, ageOK := root.Files[0].Quantity(FileAge)
	g.Expect(ageOK).To(BeFalse())

	_, addedOK := root.Files[0].Quantity(TotalLinesAdded)
	g.Expect(addedOK).To(BeFalse())

	_, densityOK := root.Files[0].Measure(CommitDensity)
	g.Expect(densityOK).To(BeFalse())
}

func TestFileFreshnessProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	root := buildTree(dir, "old.go", "new.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(Succeed())

	// old.go was committed at 2024-01-01 and never modified — should have freshness > 0
	freshOld, ok := root.Files[0].Quantity(FileFreshness)
	g.Expect(ok).To(BeTrue())
	g.Expect(freshOld).To(BeNumerically(">", 0), "old.go last modified 2024-01-01 should have freshness > 0")

	// new.go was just committed — should be very fresh (small number)
	freshNew, ok := root.Files[1].Quantity(FileFreshness)
	g.Expect(ok).To(BeTrue())
	g.Expect(freshNew).To(BeNumerically(">=", 0))

	// old.go should be staler than new.go (higher freshness = more days since last change)
	g.Expect(freshOld).To(BeNumerically(">", freshNew))
}

func TestAuthorCountProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	root := buildTree(dir, "shared.go", "old.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(Succeed())

	// shared.go: 2 authors (Alice + Bob)
	count, ok := root.Files[0].Quantity(AuthorCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(count).To(Equal(int64(2)))

	// old.go: 1 author (Alice)
	count, ok = root.Files[1].Quantity(AuthorCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(count).To(Equal(int64(1)))
}

func TestGitProviderNotAGitRepo(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	root := buildTree(dir, "file.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(MatchError(ContainSubstring("git")))
}

// TestGitProviderEmptyRepoNoHistory verifies the loader returns a clear error
// when the repository exists but has no commits — the cache cannot be primed,
// so silently producing zero metrics would cascade into confusing downstream
// failures.
func TestGitProviderEmptyRepoNoHistory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	g.Expect(err).NotTo(HaveOccurred(), "git init failed: %s", out)

	_ = os.WriteFile(filepath.Join(dir, "file.go"), []byte("package main\n"), 0o600)
	root := buildTree(dir, "file.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(MatchError(ContainSubstring("git history")))
}

// TestGitProviderNoTrackedFiles verifies the loader returns a clear error when
// the repository has history but none of the scanned files are tracked — the
// per-file metrics would all be silently absent without this guard.
func TestGitProviderNoTrackedFiles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)

	// File exists in the working tree but was never committed.
	_ = os.WriteFile(filepath.Join(dir, "ephemeral.go"), []byte("package main\n"), 0o600)
	root := buildTree(dir, "ephemeral.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(MatchError(ContainSubstring("no metrics")))
}

// TestCommitDataCacheConsistency verifies that running all three git metrics on
// the same file produces consistent results, confirming they share cached data.
func TestCommitDataCacheConsistency(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	root := buildTree(dir, "shared.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(Succeed())

	// All three metrics should be populated for shared.go.
	_, ageOk := root.Files[0].Quantity(FileAge)
	g.Expect(ageOk).To(BeTrue(), "file-age should be set")

	_, freshnessOk := root.Files[0].Quantity(FileFreshness)
	g.Expect(freshnessOk).To(BeTrue(), "file-freshness should be set")

	count, countOk := root.Files[0].Quantity(AuthorCount)
	g.Expect(countOk).To(BeTrue(), "author-count should be set")

	// shared.go has commits from both Alice and Bob.
	g.Expect(count).To(Equal(int64(2)), "shared.go should have 2 authors")
}

func TestFileAgeProviderMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	def, ok := providerDefs[FileAge]
	g.Expect(ok).To(BeTrue())
	g.Expect(def.kind).To(Equal(metric.Quantity))
	g.Expect(def.description).NotTo(BeEmpty())
	g.Expect(def.defaultPalette).NotTo(BeEmpty())
}

func TestFileFreshnessProviderMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	def, ok := providerDefs[FileFreshness]
	g.Expect(ok).To(BeTrue())
	g.Expect(def.kind).To(Equal(metric.Quantity))
	g.Expect(def.description).NotTo(BeEmpty())
	g.Expect(def.defaultPalette).NotTo(BeEmpty())
}

func TestAuthorCountProviderMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	def, ok := providerDefs[AuthorCount]
	g.Expect(ok).To(BeTrue())
	g.Expect(def.kind).To(Equal(metric.Quantity))
	g.Expect(def.description).NotTo(BeEmpty())
	g.Expect(def.defaultPalette).NotTo(BeEmpty())
}

// setupSubdirRepo creates a git repo with a file inside a subdirectory,
// committed at a fixed old date. It returns the subdirectory path.
func setupSubdirRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()

		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper
		cmd.Dir = dir

		cmd.Env = append(
			os.Environ(),
			"GIT_AUTHOR_NAME=Alice",
			"GIT_AUTHOR_EMAIL=alice@example.com",
			"GIT_COMMITTER_NAME=Alice",
			"GIT_COMMITTER_EMAIL=alice@example.com",
		)

		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %s\n%s", args, err, out)
		}
	}

	run("git", "init")
	run("git", "config", "user.name", "Alice")
	run("git", "config", "user.email", "alice@example.com")

	sub := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %s", err)
	}

	_ = os.WriteFile(filepath.Join(sub, "code.go"), []byte("package code\n"), 0o600)

	run("git", "add", ".")
	run("git", "commit", "-m", "initial commit", "--date=2024-01-01T00:00:00+00:00")

	return sub
}

func TestFileAgeProvider_SubdirectoryScanning(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	subdir := setupSubdirRepo(t)
	root := buildTree(subdir, "code.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(Succeed())

	age, ok := root.Files[0].Quantity(FileAge)
	g.Expect(ok).To(BeTrue(), "file-age metric should be set for file in subdirectory")
	g.Expect(age).To(BeNumerically(">", 0), "file committed 2024-01-01 should have age > 0")
}

func TestFileFreshnessProvider_SubdirectoryScanning(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	subdir := setupSubdirRepo(t)
	root := buildTree(subdir, "code.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(Succeed())

	freshness, ok := root.Files[0].Quantity(FileFreshness)
	g.Expect(ok).To(BeTrue(), "file-freshness metric should be set for file in subdirectory")
	g.Expect(freshness).To(BeNumerically(">", 0), "file committed 2024-01-01 should have freshness > 0")
}

func TestAuthorCountProvider_SubdirectoryScanning(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	subdir := setupSubdirRepo(t)
	root := buildTree(subdir, "code.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(Succeed())

	count, ok := root.Files[0].Quantity(AuthorCount)
	g.Expect(ok).To(BeTrue(), "author-count metric should be set for file in subdirectory")
	g.Expect(count).To(Equal(int64(1)), "code.go should have 1 author (Alice)")
}

// setupMergeRepo creates a git repo where main has two files, stable.go
// is modified once on main, and a feature branch modifies only active.go
// before being merged back. The modification of stable.go on main gives
// go-git a clear commit that's NOT TREESAME for stable.go, ensuring the
// commit is returned even with history simplification.
func setupMergeRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()

		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper
		cmd.Dir = dir

		cmd.Env = append(
			os.Environ(),
			"GIT_AUTHOR_NAME=Alice",
			"GIT_AUTHOR_EMAIL=alice@example.com",
			"GIT_COMMITTER_NAME=Alice",
			"GIT_COMMITTER_EMAIL=alice@example.com",
		)

		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %s\n%s", args, err, out)
		}
	}

	// Commit 1: create both files (backdated).
	run("git", "init", "-b", "main")
	run("git", "config", "user.name", "Alice")
	run("git", "config", "user.email", "alice@example.com")

	_ = os.WriteFile(filepath.Join(dir, "stable.go"), []byte("package stable\n"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "active.go"), []byte("package active\n"), 0o600)

	run("git", "add", ".")
	run("git", "commit", "-m", "initial commit", "--date=2024-01-01T00:00:00+00:00")

	// Commit 2 (on main): modify stable.go at a known date.
	_ = os.WriteFile(filepath.Join(dir, "stable.go"), []byte("package stable\n// updated\n"), 0o600)

	run("git", "add", "stable.go")
	run("git", "commit", "-m", "update stable", "--date=2024-06-01T00:00:00+00:00")

	// Create a feature branch that modifies only active.go.
	run("git", "checkout", "-b", "feature")

	_ = os.WriteFile(filepath.Join(dir, "active.go"), []byte("package active\n// feature\n"), 0o600)

	run("git", "add", "active.go")
	run("git", "commit", "-m", "feature change", "--date=2025-12-01T00:00:00+00:00")

	// Merge back to main — creates a merge commit that includes stable.go
	// in its tree but doesn't modify it.
	run("git", "checkout", "main")
	run("git", "merge", "feature", "--no-ff", "-m", "merge feature")

	return dir
}

// setupMergeChurnRepo creates a merge where merge.go changes on both branches
// and the resolved merge content differs from either parent.
func setupMergeChurnRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()

		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper
		cmd.Dir = dir

		cmd.Env = append(
			os.Environ(),
			"GIT_AUTHOR_NAME=Alice",
			"GIT_AUTHOR_EMAIL=alice@example.com",
			"GIT_COMMITTER_NAME=Alice",
			"GIT_COMMITTER_EMAIL=alice@example.com",
		)

		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %s\n%s", args, err, out)
		}
	}

	write := func(contents string) {
		t.Helper()

		if err := os.WriteFile(filepath.Join(dir, "merge.go"), []byte(contents), 0o600); err != nil {
			t.Fatalf("write merge.go: %s", err)
		}
	}

	run("git", "init", "-b", "main")
	run("git", "config", "user.name", "Alice")
	run("git", "config", "user.email", "alice@example.com")

	write("base-one\nbase-two\nbase-three\nbase-four\nbase-five\nbase-six\nbase-seven\n")
	run("git", "add", "merge.go")
	run("git", "commit", "-m", "initial")

	run("git", "checkout", "-b", "feature")
	write("base-one\nbase-two\nbase-three\nbase-four\nfeature-five\nfeature-six\nbase-seven\n")
	run("git", "add", "merge.go")
	run("git", "commit", "-m", "feature changes")

	run("git", "checkout", "main")
	write("main-one\nbase-two\nbase-three\nbase-four\nbase-five\nbase-six\nbase-seven\n")
	run("git", "add", "merge.go")
	run("git", "commit", "-m", "main changes")

	run("git", "merge", "feature", "--no-ff", "--no-commit")
	write("main-one\nbase-two\nbase-three\nbase-four\nresolved-five\nresolved-six\nbase-seven\nmerge-eight\n")
	run("git", "add", "merge.go")
	run("git", "commit", "-m", "resolve merge")

	return dir
}

// TestFileFreshness_MergeCommitDoesNotPollute verifies that a merge commit
// touching stable.go's tree entry (but not its content) doesn't update the
// freshness timestamp for stable.go. This was the root cause of #114.
func TestFileFreshness_MergeCommitDoesNotPollute(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupMergeRepo(t)
	root := buildTree(dir, "stable.go", "active.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(Succeed())

	// stable.go was last truly modified at 2024-06-01. Its freshness (days
	// since last real change) should be > 300. Without the fix, the merge
	// commit's timestamp (today) would pollute this to ~0.
	freshStable, ok := root.Files[0].Quantity(FileFreshness)
	g.Expect(ok).To(BeTrue(), "file-freshness should be set for stable.go")
	g.Expect(freshStable).To(BeNumerically(">", 300),
		"stable.go last modified 2024-06-01 should have high freshness (days since change)")

	// active.go was modified at 2025-12-01 — should have a moderate freshness.
	freshActive, ok := root.Files[1].Quantity(FileFreshness)
	g.Expect(ok).To(BeTrue(), "file-freshness should be set for active.go")
	g.Expect(freshActive).To(BeNumerically(">", 0),
		"active.go last modified 2025-12-01 should have freshness > 0")

	// stable.go must be staler than active.go.
	g.Expect(freshStable).To(BeNumerically(">", freshActive),
		"stable.go should be staler than active.go")
}

// TestFileAge_MergeCommitDoesNotPollute verifies that a merge commit doesn't
// shift the oldest timestamp for files it didn't modify.
func TestFileAge_MergeCommitDoesNotPollute(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupMergeRepo(t)
	root := buildTree(dir, "stable.go", "active.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(Succeed())

	// Both files were created at 2024-01-01 — same age.
	ageStable, ok := root.Files[0].Quantity(FileAge)
	g.Expect(ok).To(BeTrue())
	g.Expect(ageStable).To(BeNumerically(">", 300))

	ageActive, ok := root.Files[1].Quantity(FileAge)
	g.Expect(ok).To(BeTrue())
	g.Expect(ageActive).To(BeNumerically(">", 300))
}

// TestAuthorCount_MergeCommitDoesNotPollute verifies that the merge commit
// author is not counted for files the merge didn't actually modify.
func TestAuthorCount_MergeCommitDoesNotPollute(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupMergeRepo(t)
	root := buildTree(dir, "stable.go", "active.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(Succeed())

	// stable.go was only committed by Alice — the merge didn't change it.
	count, ok := root.Files[0].Quantity(AuthorCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(count).To(Equal(int64(1)), "stable.go should have 1 author")

	// active.go was committed by Alice initially — the feature branch commit
	// was also by Alice (our test setup uses Alice for all commits).
	countActive, ok := root.Files[1].Quantity(AuthorCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(countActive).To(Equal(int64(1)), "active.go should have 1 author (all commits by Alice)")
}

// TestFileFreshnessEqualsAgeForSingleCommit verifies that for a file with
// exactly one commit, file-freshness equals file-age (both measure days since
// the same single commit).
func TestFileFreshnessEqualsAgeForSingleCommit(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupSubdirRepo(t) // code.go committed once at 2024-01-01
	root := buildTree(dir, "code.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(Succeed())

	age, ageOk := root.Files[0].Quantity(FileAge)
	freshness, freshOk := root.Files[0].Quantity(FileFreshness)

	g.Expect(ageOk).To(BeTrue())
	g.Expect(freshOk).To(BeTrue())
	g.Expect(age).To(Equal(freshness),
		"single-commit file should have identical age and freshness")
}

func TestCommitCountProviderMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	def, ok := providerDefs[CommitCount]
	g.Expect(ok).To(BeTrue())
	g.Expect(def.kind).To(Equal(metric.Quantity))
	g.Expect(def.description).NotTo(BeEmpty())
	g.Expect(def.defaultPalette).NotTo(BeEmpty())
}

func TestCommitCountProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	// old.go: 1 commit (initial). shared.go: 2 commits (initial + bob's update).
	root := buildTree(dir, "old.go", "shared.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(Succeed())

	countOld, ok := root.Files[0].Quantity(CommitCount)
	g.Expect(ok).To(BeTrue(), "commit-count should be set for old.go")
	g.Expect(countOld).To(Equal(int64(1)), "old.go was committed once")

	countShared, ok := root.Files[1].Quantity(CommitCount)
	g.Expect(ok).To(BeTrue(), "commit-count should be set for shared.go")
	g.Expect(countShared).To(Equal(int64(2)), "shared.go was committed twice (initial + bob's update)")
}

// TestCommitCount_MergeCommitDoesNotPollute verifies that a merge commit that
// doesn't modify a file is not counted as a commit for that file.
func TestCommitCount_MergeCommitDoesNotPollute(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupMergeRepo(t)
	// stable.go: initial + update on main = 2 real commits; merge doesn't modify it.
	// active.go: initial + feature change = 2 real commits; merge doesn't add a new one.
	root := buildTree(dir, "stable.go", "active.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(Succeed())

	countStable, ok := root.Files[0].Quantity(CommitCount)
	g.Expect(ok).To(BeTrue(), "commit-count should be set for stable.go")
	g.Expect(countStable).To(Equal(int64(2)),
		"stable.go: initial commit + update on main; merge should not be counted")

	countActive, ok := root.Files[1].Quantity(CommitCount)
	g.Expect(ok).To(BeTrue(), "commit-count should be set for active.go")
	g.Expect(countActive).To(Equal(int64(2)),
		"active.go: initial commit + feature branch change; merge should not be counted")
}

func TestCommitCountProvider_SubdirectoryScanning(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	subdir := setupSubdirRepo(t)
	root := buildTree(subdir, "code.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(Succeed())

	count, ok := root.Files[0].Quantity(CommitCount)
	g.Expect(ok).To(BeTrue(), "commit-count metric should be set for file in subdirectory")
	g.Expect(count).To(Equal(int64(1)), "code.go was committed once")
}

func TestIsGitMetric_NewMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(IsGitMetric(TotalLinesAdded)).To(BeTrue())
	g.Expect(IsGitMetric(TotalLinesRemoved)).To(BeTrue())
	g.Expect(IsGitMetric(CommitDensity)).To(BeTrue())
}

func TestTotalLinesAddedProviderMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	def, ok := providerDefs[TotalLinesAdded]
	g.Expect(ok).To(BeTrue())
	g.Expect(def.kind).To(Equal(metric.Quantity))
	g.Expect(def.description).NotTo(BeEmpty())
	g.Expect(def.defaultPalette).NotTo(BeEmpty())
}

func TestTotalLinesRemovedProviderMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	def, ok := providerDefs[TotalLinesRemoved]
	g.Expect(ok).To(BeTrue())
	g.Expect(def.kind).To(Equal(metric.Quantity))
	g.Expect(def.description).NotTo(BeEmpty())
	g.Expect(def.defaultPalette).NotTo(BeEmpty())
}

func TestCommitDensityProviderMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	def, ok := providerDefs[CommitDensity]
	g.Expect(ok).To(BeTrue())
	g.Expect(def.kind).To(Equal(metric.Measure))
	g.Expect(def.description).NotTo(BeEmpty())
	g.Expect(def.defaultPalette).NotTo(BeEmpty())
}

// setupDiffRepo creates a git repo with a file that has measurable additions
// and removals across multiple commits.
func setupDiffRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()

		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper
		cmd.Dir = dir

		cmd.Env = append(
			os.Environ(),
			"GIT_AUTHOR_NAME=Alice",
			"GIT_AUTHOR_EMAIL=alice@example.com",
			"GIT_COMMITTER_NAME=Alice",
			"GIT_COMMITTER_EMAIL=alice@example.com",
		)

		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %s\n%s", args, err, out)
		}
	}

	run("git", "init")
	run("git", "config", "user.name", "Alice")
	run("git", "config", "user.email", "alice@example.com")

	// Commit 1: create file with 3 lines (creation commit — excluded from stats)
	_ = os.WriteFile(filepath.Join(dir, "churn.go"),
		[]byte("line1\nline2\nline3\n"), 0o600)

	run("git", "add", ".")
	run("git", "commit", "-m", "initial", "--date=2024-01-01T00:00:00+00:00")

	// Commit 2: add 2 lines (total: 5 lines)
	_ = os.WriteFile(filepath.Join(dir, "churn.go"),
		[]byte("line1\nline2\nline3\nline4\nline5\n"), 0o600)

	run("git", "add", ".")
	run("git", "commit", "-m", "add lines", "--date=2024-02-01T00:00:00+00:00")

	// Commit 3: remove 1 line, add 1 line (replace line2 with lineX)
	_ = os.WriteFile(filepath.Join(dir, "churn.go"),
		[]byte("line1\nlineX\nline3\nline4\nline5\n"), 0o600)

	run("git", "add", ".")
	run("git", "commit", "-m", "modify", "--date=2024-03-01T00:00:00+00:00")

	// Also create a stable file with no changes after creation
	_ = os.WriteFile(filepath.Join(dir, "stable.go"),
		[]byte("package stable\n"), 0o600)

	run("git", "add", ".")
	run("git", "commit", "-m", "add stable", "--date=2024-03-15T00:00:00+00:00")

	return dir
}

// setupMultiFileDiffRepo adds a commit which changes two tracked files. It
// exercises reuse of one commit tree diff for both files' churn statistics.
func setupMultiFileDiffRepo(t *testing.T) string {
	t.Helper()

	dir := setupDiffRepo(t)
	write := func(name, content string) {
		t.Helper()
		g := NewGomegaWithT(t)
		g.Expect(os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600)).To(Succeed())
	}

	run := func(args ...string) {
		t.Helper()

		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper
		cmd.Dir = dir

		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("command %v failed: %s\n%s", args, err, out)
		}
	}

	write("other.go", "one\ntwo\n")
	run("git", "add", "other.go")
	run("git", "commit", "-m", "add other")

	write("churn.go", "line1\nlineX\nline3\nline4\nline5\nline6\n")
	write("other.go", "one\ntwo\nthree\n")
	run("git", "add", "churn.go", "other.go")
	run("git", "commit", "-m", "change both")

	return dir
}

func TestLoadGitMetrics_ChurnForMultiFileCommit(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupMultiFileDiffRepo(t)
	root := buildTree(dir, "churn.go", "other.go")

	resetService()
	g.Expect(loadGitMetrics(
		root,
		[]metric.Name{TotalLinesAdded, TotalLinesRemoved},
		nil,
	)).To(Succeed())

	churnAdded, ok := root.Files[0].Quantity(TotalLinesAdded)
	g.Expect(ok).To(BeTrue())
	g.Expect(churnAdded).To(Equal(int64(4)))

	churnRemoved, ok := root.Files[0].Quantity(TotalLinesRemoved)
	g.Expect(ok).To(BeTrue())
	g.Expect(churnRemoved).To(Equal(int64(1)))

	otherAdded, ok := root.Files[1].Quantity(TotalLinesAdded)
	g.Expect(ok).To(BeTrue())
	g.Expect(otherAdded).To(Equal(int64(1)))

	otherRemoved, ok := root.Files[1].Quantity(TotalLinesRemoved)
	g.Expect(ok).To(BeTrue())
	g.Expect(otherRemoved).To(Equal(int64(0)))

	_, countOK := root.Files[0].Quantity(CommitCount)
	g.Expect(countOK).To(BeFalse())
}

func TestTrackedChangesInCommit_RetainsChangesForEachTrackedPath(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupMultiFileDiffRepo(t)

	resetService()

	s, err := getService(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("expected git repository service")
	}

	head, err := s.repo.Head()
	g.Expect(err).NotTo(HaveOccurred())

	commit, err := s.repo.CommitObject(head.Hash())
	g.Expect(err).NotTo(HaveOccurred())

	changes := trackedChangesInCommit(commit, map[string]bool{
		"churn.go": true,
		"other.go": true,
	})
	g.Expect(changes).To(HaveLen(2))

	for _, entry := range changes {
		g.Expect(entry.change).NotTo(BeNil())
	}
}

func TestTrackedChangesInMergeCommit_LeavesChurnChangeNilWhenFirstParentFails(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := memory.NewStorage()
	storeBlob := func(contents string) plumbing.Hash {
		t.Helper()

		encoded := store.NewEncodedObject()
		encoded.SetType(plumbing.BlobObject)

		writer, err := encoded.Writer()
		g.Expect(err).NotTo(HaveOccurred())
		_, err = writer.Write([]byte(contents))
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(writer.Close()).To(Succeed())

		hash, err := store.SetEncodedObject(encoded)
		g.Expect(err).NotTo(HaveOccurred())

		return hash
	}
	storeTree := func(blobHash plumbing.Hash) plumbing.Hash {
		t.Helper()

		tree := &object.Tree{Entries: []object.TreeEntry{{
			Name: "merge.go",
			Mode: filemode.Regular,
			Hash: blobHash,
		}}}
		encoded := store.NewEncodedObject()
		g.Expect(tree.Encode(encoded)).To(Succeed())

		hash, err := store.SetEncodedObject(encoded)
		g.Expect(err).NotTo(HaveOccurred())

		return hash
	}
	storeCommit := func(treeHash plumbing.Hash, parents []plumbing.Hash) plumbing.Hash {
		t.Helper()

		commit := &object.Commit{TreeHash: treeHash, ParentHashes: parents}
		encoded := store.NewEncodedObject()
		g.Expect(commit.Encode(encoded)).To(Succeed())

		hash, err := store.SetEncodedObject(encoded)
		g.Expect(err).NotTo(HaveOccurred())

		return hash
	}

	firstParent := storeCommit(plumbing.NewHash("missing tree"), nil)
	secondParent := storeCommit(storeTree(storeBlob("second parent")), nil)
	mergeHash := storeCommit(storeTree(storeBlob("merge")), []plumbing.Hash{
		firstParent,
		secondParent,
	})
	merge, err := object.GetCommit(store, mergeHash)
	g.Expect(err).NotTo(HaveOccurred())

	if merge == nil {
		t.Fatal("expected merge commit")
	}

	changes := trackedChangesInCommit(merge, map[string]bool{"merge.go": true})
	g.Expect(changes).To(HaveLen(1))

	if len(changes) != 1 {
		t.Fatal("expected tracked change for merge.go")
	}

	g.Expect(changes[0].path).To(Equal("merge.go"))
	g.Expect(changes[0].change).To(BeNil())
}

func TestLoadGitMetrics_AttributesMergeChurnToFirstParent(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupMergeChurnRepo(t)
	root := buildTree(dir, "merge.go")

	resetService()

	s, err := getService(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("expected git repository service")
	}

	g.Expect(s.loadGitMetrics(
		root,
		[]metric.Name{TotalLinesAdded, TotalLinesRemoved},
		nil,
	)).To(Succeed())

	added, ok := root.Files[0].Quantity(TotalLinesAdded)
	g.Expect(ok).To(BeTrue())
	// Main changes add/remove 1 line, feature changes add/remove 2 lines,
	// and the resolved merge adds 3 and removes 2 against parent 0.
	g.Expect(added).To(Equal(int64(6)))

	removed, ok := root.Files[0].Quantity(TotalLinesRemoved)
	g.Expect(ok).To(BeTrue())
	g.Expect(removed).To(Equal(int64(5)))
}

func TestGetCommitData_SeparatesMetadataAndLineStatsFlights(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	metadataFetchStarted := make(chan struct{})
	lineStatsFetchStarted := make(chan struct{})
	releaseMetadataFetch := make(chan struct{})

	var releaseOnce sync.Once

	release := func() {
		releaseOnce.Do(func() {
			close(releaseMetadataFetch)
		})
	}
	defer release()

	var fetches atomic.Int64

	s := &repoService{
		commitCache: make(map[string]*commitData),
		fetchCommitDataFn: func(string) (*commitData, error) {
			if fetches.Add(1) == 1 {
				close(metadataFetchStarted)
				<-releaseMetadataFetch

				return &commitData{hasLineStats: false}, nil
			}

			close(lineStatsFetchStarted)

			return &commitData{hasLineStats: true}, nil
		},
	}

	metadataResult := make(chan error, 1)

	go func() {
		_, err := s.getMetadataCommitData("file.go")
		metadataResult <- err
	}()

	select {
	case <-metadataFetchStarted:
	case <-time.After(time.Second):
		t.Fatal("metadata request did not start fetching")
	}

	type lineStatsResult struct {
		data *commitData
		err  error
	}

	lineStatsResultCh := make(chan lineStatsResult, 1)

	go func() {
		data, err := s.getLineStatsCommitData("file.go")
		lineStatsResultCh <- lineStatsResult{data: data, err: err}
	}()

	select {
	case <-lineStatsFetchStarted:
	case <-time.After(time.Second):
		t.Fatal("line-stat request waited on the metadata-only flight")
	}

	result := <-lineStatsResultCh
	g.Expect(result.err).NotTo(HaveOccurred())
	g.Expect(result.data).NotTo(BeNil())
	g.Expect(result.data.hasLineStats).To(BeTrue())

	release()
	g.Expect(<-metadataResult).To(Succeed())
}

func TestBulkPrewarm_UpgradesMetadataCacheForLineStats(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupDiffRepo(t)

	resetService()

	s, err := getService(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("expected git repository service")
	}

	paths := map[string]bool{"churn.go": true}
	metadata := newMetricRequirements([]metric.Name{CommitCount})
	g.Expect(s.bulkPrewarm(paths, metadata, nil)).To(Succeed())

	s.commitMu.RLock()
	cached := s.commitCache["churn.go"]
	s.commitMu.RUnlock()
	g.Expect(cached).NotTo(BeNil())

	if cached == nil {
		t.Fatal("expected cached metadata")
	}

	g.Expect(cached.hasLineStats).To(BeFalse())

	lineStats := newMetricRequirements([]metric.Name{TotalLinesAdded, TotalLinesRemoved})
	g.Expect(s.bulkPrewarm(paths, lineStats, nil)).To(Succeed())

	s.commitMu.RLock()
	cached = s.commitCache["churn.go"]
	s.commitMu.RUnlock()
	g.Expect(cached).NotTo(BeNil())

	if cached == nil {
		t.Fatal("expected cached line statistics")
	}

	g.Expect(cached.hasLineStats).To(BeTrue())
	g.Expect(cached.linesAdded).To(Equal(int64(3)))
	g.Expect(cached.linesRemoved).To(Equal(int64(1)))
}

func TestBulkPrewarm_PreservesLineStatsDuringMetadataPrewarm(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &repoService{commitCache: map[string]*commitData{
		"churn.go": {
			hasLineStats: true,
			linesAdded:   3,
			linesRemoved: 1,
		},
	}}
	metadata := newMetricRequirements([]metric.Name{CommitCount})
	s.mergeBulkPrewarmCache(map[string]*commitData{
		"churn.go": {hasLineStats: false},
	}, metadata)

	s.commitMu.RLock()
	cached := s.commitCache["churn.go"]
	s.commitMu.RUnlock()
	g.Expect(cached).NotTo(BeNil())

	if cached == nil {
		t.Fatal("expected cached line statistics")
	}

	g.Expect(cached.hasLineStats).To(BeTrue())
	g.Expect(cached.linesAdded).To(Equal(int64(3)))
	g.Expect(cached.linesRemoved).To(Equal(int64(1)))
}

func TestTotalLinesAddedProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupDiffRepo(t)
	root := buildTree(dir, "churn.go", "stable.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(Succeed())

	// churn.go: commit 2 adds 2 lines, commit 3 adds 1 line = 3 total
	added, ok := root.Files[0].Quantity(TotalLinesAdded)
	g.Expect(ok).To(BeTrue(), "total-lines-added should be set for churn.go")
	g.Expect(added).To(Equal(int64(3)), "churn.go: 2 lines in commit 2 + 1 line in commit 3")

	// stable.go: created in a non-root commit but only has the creation commit
	// (no subsequent modifications) — should be 0
	added, ok = root.Files[1].Quantity(TotalLinesAdded)
	g.Expect(ok).To(BeTrue(), "total-lines-added should be set for stable.go")
	g.Expect(added).To(Equal(int64(0)), "stable.go has no modifications after creation")
}

func TestMetricsLoaderLoadsOnlyRequestedTotalLinesAdded(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupDiffRepo(t)
	root := buildTree(dir, "churn.go")

	resetService()
	g.Expect((&metricsLoader{}).Load(root, []metric.Name{TotalLinesAdded})).To(Succeed())

	added, addedOK := root.Files[0].Quantity(TotalLinesAdded)
	g.Expect(addedOK).To(BeTrue())
	g.Expect(added).To(Equal(int64(3)))

	_, removedOK := root.Files[0].Quantity(TotalLinesRemoved)
	g.Expect(removedOK).To(BeFalse())

	_, countOK := root.Files[0].Quantity(CommitCount)
	g.Expect(countOK).To(BeFalse())
}

func TestLoadGitMetrics_RefreshesChurnAfterMetadataOnlyPrewarm(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupDiffRepo(t)
	root := buildTree(dir, "churn.go")

	resetService()

	s, err := getService(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("expected git repository service")
	}

	g.Expect(s.loadGitMetrics(root, []metric.Name{CommitCount}, nil)).To(Succeed())

	count, ok := root.Files[0].Quantity(CommitCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(count).To(Equal(int64(3)))

	g.Expect(s.loadGitMetrics(root, []metric.Name{TotalLinesAdded}, nil)).To(Succeed())

	added, ok := root.Files[0].Quantity(TotalLinesAdded)
	g.Expect(ok).To(BeTrue())
	g.Expect(added).To(Equal(int64(3)))
}

func TestTotalLinesRemovedProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupDiffRepo(t)
	root := buildTree(dir, "churn.go", "stable.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(Succeed())

	// churn.go: commit 3 removes 1 line (line2 → lineX) = 1 total
	removed, ok := root.Files[0].Quantity(TotalLinesRemoved)
	g.Expect(ok).To(BeTrue(), "total-lines-removed should be set for churn.go")
	g.Expect(removed).To(Equal(int64(1)), "churn.go: 1 line removed in commit 3")

	// stable.go: no removals
	removed, ok = root.Files[1].Quantity(TotalLinesRemoved)
	g.Expect(ok).To(BeTrue(), "total-lines-removed should be set for stable.go")
	g.Expect(removed).To(Equal(int64(0)), "stable.go has no modifications")
}

func TestCommitDensityProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupDiffRepo(t)
	root := buildTree(dir, "churn.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(Succeed())

	// churn.go: 3 commits, age > 1 month. Density = 3 / age_months.
	density, ok := root.Files[0].Measure(CommitDensity)
	g.Expect(ok).To(BeTrue(), "commit-density should be set for churn.go")
	g.Expect(density).To(BeNumerically(">", 0), "commit-density should be positive")
	// With ~18 months of age: 3/18 ≈ 0.17. Allow a wide range for time passing.
	g.Expect(density).To(BeNumerically("<", 3), "commit-density should be reasonable")
}

func TestCommitDensityProvider_YoungFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t) // new.go was just committed
	root := buildTree(dir, "new.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(Succeed())

	// new.go: 1 commit, age < 1 month → clamped to 1 month. Density = 1/1 = 1.0
	density, ok := root.Files[0].Measure(CommitDensity)
	g.Expect(ok).To(BeTrue(), "commit-density should be set for new.go")
	g.Expect(density).To(BeNumerically("~", 1.0, 0.01),
		"young file with 1 commit should have density ≈ 1.0")
}

func TestTotalLinesAdded_NotAGitRepo(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	root := buildTree(dir, "file.go")

	resetService()
	g.Expect(loadAllFileMetrics(root)).To(MatchError(ContainSubstring("git")))
}
