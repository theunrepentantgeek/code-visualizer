package stages

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
)

func TestResolveSourceUsesHistoricalTreeForUntil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	runGitForSourceTest(t, dir, "init", "-b", "main")
	runGitForSourceTest(t, dir, "config", "user.name", "Test")
	runGitForSourceTest(t, dir, "config", "user.email", "test@example.com")
	g.Expect(os.WriteFile(filepath.Join(dir, "old.txt"), []byte("historical\n"), 0o600)).To(Succeed())
	runGitForSourceTest(t, dir, "add", ".")
	runGitForSourceTest(t, dir, "commit", "-m", "historical")
	oldCommit := runGitForSourceTest(t, dir, "rev-parse", "HEAD")
	g.Expect(os.Remove(filepath.Join(dir, "old.txt"))).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "new.txt"), []byte("current\n"), 0o600)).To(Succeed())

	state := &CommonState{
		TargetPath: dir,
		Flags:      &Flags{HistoryRange: git.HistoryRange{Until: "sha:" + oldCommit}},
	}
	g.Expect(ResolveSource(state)).To(Succeed())
	g.Expect(ScanFilesystem(state)).To(Succeed())
	g.Expect(state.Root.Files).To(HaveLen(1))
	g.Expect(state.Root.Files[0].Name).To(Equal("old.txt"))
	data, err := state.Root.Files[0].ReadAll()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(data)).To(Equal("historical\n"))
}

func TestResolveSourceUsesHistoricalSubtreeDeletedFromWorkingTree(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	runGitForSourceTest(t, dir, "init", "-b", "main")
	runGitForSourceTest(t, dir, "config", "user.name", "Test")
	runGitForSourceTest(t, dir, "config", "user.email", "test@example.com")
	oldDir := filepath.Join(dir, "old")
	g.Expect(os.Mkdir(oldDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(oldDir, "file.txt"), []byte("historical\n"), 0o600)).To(Succeed())
	runGitForSourceTest(t, dir, "add", ".")
	runGitForSourceTest(t, dir, "commit", "-m", "historical")
	oldCommit := runGitForSourceTest(t, dir, "rev-parse", "HEAD")

	g.Expect(os.RemoveAll(oldDir)).To(Succeed())

	state := &CommonState{
		TargetPath: oldDir,
		Output:     filepath.Join(dir, "out.png"),
		Flags:      &Flags{HistoryRange: git.HistoryRange{Until: "sha:" + oldCommit}},
	}
	g.Expect(ValidatePaths(state)).To(Succeed())
	g.Expect(ResolveSource(state)).To(Succeed())
	g.Expect(ScanFilesystem(state)).To(Succeed())
	g.Expect(state.Root.Files).To(HaveLen(1))
	g.Expect(state.Root.Files[0].Name).To(Equal("file.txt"))
}

func runGitForSourceTest(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}

	return string(bytes.TrimSpace(out))
}
