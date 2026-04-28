package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
)

func setupTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()

		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper
		cmd.Dir = dir

		cmd.Env = append(os.Environ(),
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

		cmd.Env = append(os.Environ(),
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

	p := &FileAgeProvider{}
	g.Expect(p.Name()).To(Equal(FileAge))
	g.Expect(p.Kind()).To(Equal(metric.Quantity))

	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

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

func TestFileFreshnessProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	root := buildTree(dir, "old.go", "new.go")

	resetService()

	p := &FileFreshnessProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

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

	p := &AuthorCountProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

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

	p := &FileAgeProvider{}
	err := p.Load(root)
	g.Expect(err).To(MatchError(ContainSubstring("git")))
}

// TestCommitDataCacheConsistency verifies that running all three git metrics on
// the same file produces consistent results, confirming they share cached data.
func TestCommitDataCacheConsistency(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	root := buildTree(dir, "shared.go")

	resetService()

	// Run all three git metric providers on the same file.
	fileAgeP := &FileAgeProvider{}
	err := fileAgeP.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	fileFreshnessP := &FileFreshnessProvider{}
	err = fileFreshnessP.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	authorCountP := &AuthorCountProvider{}
	err = authorCountP.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

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

	p := &FileAgeProvider{}
	g.Expect(p.Name()).To(Equal(FileAge))
	g.Expect(p.Kind()).To(Equal(metric.Quantity))
	g.Expect(p.Description()).NotTo(BeEmpty())
	g.Expect(p.DefaultPalette()).NotTo(BeEmpty())
	g.Expect(p.Dependencies()).To(BeNil())
}

func TestFileFreshnessProviderMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := &FileFreshnessProvider{}
	g.Expect(p.Name()).To(Equal(FileFreshness))
	g.Expect(p.Kind()).To(Equal(metric.Quantity))
	g.Expect(p.Description()).NotTo(BeEmpty())
	g.Expect(p.DefaultPalette()).NotTo(BeEmpty())
	g.Expect(p.Dependencies()).To(BeNil())
}

func TestAuthorCountProviderMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := &AuthorCountProvider{}
	g.Expect(p.Name()).To(Equal(AuthorCount))
	g.Expect(p.Kind()).To(Equal(metric.Quantity))
	g.Expect(p.Description()).NotTo(BeEmpty())
	g.Expect(p.DefaultPalette()).NotTo(BeEmpty())
	g.Expect(p.Dependencies()).To(BeNil())
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

		cmd.Env = append(os.Environ(),
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

	p := &FileAgeProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

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

	p := &FileFreshnessProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

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

	p := &AuthorCountProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

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

		cmd.Env = append(os.Environ(),
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

// TestFileFreshness_MergeCommitDoesNotPollute verifies that a merge commit
// touching stable.go's tree entry (but not its content) doesn't update the
// freshness timestamp for stable.go. This was the root cause of #114.
func TestFileFreshness_MergeCommitDoesNotPollute(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupMergeRepo(t)
	root := buildTree(dir, "stable.go", "active.go")

	resetService()

	p := &FileFreshnessProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

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

	p := &FileAgeProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

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

	p := &AuthorCountProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

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

	ageP := &FileAgeProvider{}
	g.Expect(ageP.Load(root)).To(Succeed())

	freshP := &FileFreshnessProvider{}
	g.Expect(freshP.Load(root)).To(Succeed())

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

p := &CommitCountProvider{}
g.Expect(p.Name()).To(Equal(CommitCount))
g.Expect(p.Kind()).To(Equal(metric.Quantity))
g.Expect(p.Description()).NotTo(BeEmpty())
g.Expect(p.DefaultPalette()).NotTo(BeEmpty())
g.Expect(p.Dependencies()).To(BeNil())
}

func TestCommitCountProvider(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

dir := setupTestGitRepo(t)
// old.go: 1 commit (initial). shared.go: 2 commits (initial + bob's update).
root := buildTree(dir, "old.go", "shared.go")

resetService()

p := &CommitCountProvider{}
g.Expect(p.Name()).To(Equal(CommitCount))
g.Expect(p.Kind()).To(Equal(metric.Quantity))

err := p.Load(root)
g.Expect(err).NotTo(HaveOccurred())

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

p := &CommitCountProvider{}
err := p.Load(root)
g.Expect(err).NotTo(HaveOccurred())

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

p := &CommitCountProvider{}
err := p.Load(root)
g.Expect(err).NotTo(HaveOccurred())

count, ok := root.Files[0].Quantity(CommitCount)
g.Expect(ok).To(BeTrue(), "commit-count metric should be set for file in subdirectory")
g.Expect(count).To(Equal(int64(1)), "code.go was committed once")
}
