package scan

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

// setupTestGitRepo creates a temporary git repo with known commits for testing.
// Returns the path to the repo root.
func setupTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()

		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper with hardcoded args
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

		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper with hardcoded args
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

	// Init repo
	run("git", "init")
	run("git", "config", "user.name", "Alice")
	run("git", "config", "user.email", "alice@example.com")

	// First commit: alice creates old.go and shared.go
	_ = os.WriteFile(filepath.Join(dir, "old.go"), []byte("package main\n"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "shared.go"), []byte("package shared\n"), 0o600)

	run("git", "add", ".")
	run("git", "commit", "-m", "initial commit", "--date=2024-01-01T00:00:00+00:00")

	// Second commit: bob modifies shared.go
	_ = os.WriteFile(filepath.Join(dir, "shared.go"), []byte("package shared\n// updated by bob\n"), 0o600)

	runAs("Bob", "bob@example.com", "git", "add", "shared.go")
	runAs("Bob", "bob@example.com", "git", "commit", "-m", "bob update", "--date=2025-06-15T00:00:00+00:00")

	// Third commit: alice adds new.go
	_ = os.WriteFile(filepath.Join(dir, "new.go"), []byte("package new\nfunc New() {}\n"), 0o600)

	run("git", "add", "new.go")
	run("git", "commit", "-m", "add new.go")

	// Add an untracked file
	_ = os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("not tracked\n"), 0o600)

	return dir
}

func TestIsGitRepo_Yes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := setupTestGitRepo(t)

	isGit, err := IsGitRepo(dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(isGit).To(BeTrue())
}

func TestIsGitRepo_No(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := t.TempDir()

	isGit, err := IsGitRepo(dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(isGit).To(BeFalse())
}

func TestIsGitRepo_NestedInside(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := setupTestGitRepo(t)

	subDir := filepath.Join(dir, "subdir")
	_ = os.Mkdir(subDir, 0o755)

	isGit, err := IsGitRepo(subDir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(isGit).To(BeTrue())
}

func TestFileAge(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := setupTestGitRepo(t)

	info, err := NewGitInfo(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if info == nil {
		return
	}

	// old.go was first committed on 2024-01-01
	age, err := info.FileAge("old.go")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(age).NotTo(BeNil())

	if age != nil {
		// Should be > 2 years old
		g.Expect(*age).To(BeNumerically(">", 2*365*24*time.Hour))
	}
}

func TestFileFreshness(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := setupTestGitRepo(t)

	info, err := NewGitInfo(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if info == nil {
		return
	}

	// new.go was just committed (the third commit is "now"-ish)
	freshness, err := info.FileFreshness("new.go")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(freshness).NotTo(BeNil())

	if freshness != nil {
		// Should be very recent (less than 1 minute)
		g.Expect(*freshness).To(BeNumerically("<", time.Minute))
	}
}

func TestAuthorCount(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := setupTestGitRepo(t)

	info, err := NewGitInfo(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if info == nil {
		return
	}

	// shared.go was committed by Alice and Bob
	count, err := info.AuthorCount("shared.go")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(count).NotTo(BeNil())

	if count != nil {
		g.Expect(*count).To(Equal(2))
	}

	// old.go was only committed by Alice
	count, err = info.AuthorCount("old.go")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(count).NotTo(BeNil())

	if count != nil {
		g.Expect(*count).To(Equal(1))
	}
}

func TestBinaryDetection(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := setupTestGitRepo(t)

	info, err := NewGitInfo(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if info == nil {
		return
	}

	// old.go is a text file
	isBin := info.IsBinary("old.go")
	g.Expect(isBin).To(BeFalse())
}

func TestUntrackedFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := setupTestGitRepo(t)

	info, err := NewGitInfo(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if info == nil {
		return
	}

	// untracked.txt has no git history
	age, err := info.FileAge("untracked.txt")
	g.Expect(err).To(MatchError(ErrUntracked))
	g.Expect(age).To(BeNil())

	freshness, err := info.FileFreshness("untracked.txt")
	g.Expect(err).To(MatchError(ErrUntracked))
	g.Expect(freshness).To(BeNil())

	count, err := info.AuthorCount("untracked.txt")
	g.Expect(err).To(MatchError(ErrUntracked))
	g.Expect(count).To(BeNil())
}
