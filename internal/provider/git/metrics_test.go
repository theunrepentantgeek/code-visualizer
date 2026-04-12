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

	// new.go was just committed — should be very fresh (small number)
	freshNew, ok := root.Files[1].Quantity(FileFreshness)
	g.Expect(ok).To(BeTrue())
	g.Expect(freshNew).To(BeNumerically(">=", 0))
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
	g.Expect(count).To(Equal(2))

	// old.go: 1 author (Alice)
	count, ok = root.Files[1].Quantity(AuthorCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(count).To(Equal(1))
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
