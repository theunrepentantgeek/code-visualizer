package stages

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/progress"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/golang"
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
	g.Expect(ScanFilesystem(state, context.Background(), newTestSink(progress.WorkObservations))).To(Succeed())
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
		Flags: &Flags{
			ChangedOnly:  true,
			HistoryRange: git.HistoryRange{Until: "sha:" + oldCommit},
		},
		Requested: RequestedMetrics{BaseMetrics: []metric.Name{git.CommitCount}},
	}
	g.Expect(ValidatePaths(state)).To(Succeed())
	g.Expect(ResolveSource(state)).To(Succeed())
	g.Expect(ScanFilesystem(state, context.Background(), newTestSink(progress.WorkObservations))).To(Succeed())
	g.Expect(FilterChangedOnly(state, context.Background())).To(Succeed())
	g.Expect(CheckGitRequirement(state)).To(Succeed())
	g.Expect(LoadGitHistory(state, context.Background(), newTestSink(progress.WorkCommits))).To(Succeed())
	g.Expect(RunProviders(state, context.Background(), newTestSink(progress.WorkObservations))).To(Succeed())
	g.Expect(state.Root.Files).To(HaveLen(1))
	g.Expect(state.Root.Files[0].Name).To(Equal("file.txt"))
	count, ok := state.Root.Files[0].Quantity(git.CommitCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(count).To(Equal(int64(1)))
}

//nolint:paralleltest // mutates the global provider registry
func TestNonGitScopedGoProviderFindsParentModule(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "pkg")
	g.Expect(os.Mkdir(pkgDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(dir, "go.mod"),
		[]byte("module example.com/project\n\ngo 1.26\n"),
		0o600,
	)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(pkgDir, "main.go"),
		[]byte("package pkg\n\nimport \"example.com/project/internal/value\"\n"),
		0o600,
	)).To(Succeed())

	provider.ResetBaseRegistryForTesting()
	golang.Register()
	t.Cleanup(func() {
		provider.ResetBaseRegistryForTesting()
		filesystem.Register()
		git.Register()
	})

	state := &CommonState{
		TargetPath: pkgDir,
		Flags:      &Flags{},
		Requested: RequestedMetrics{
			BaseMetrics: []metric.Name{metric.Name("internal.imports")},
		},
	}
	g.Expect(ResolveSource(state)).To(Succeed())
	g.Expect(ScanFilesystem(state, context.Background(), newTestSink(progress.WorkObservations))).To(Succeed())
	g.Expect(RunProviders(state, context.Background(), newTestSink(progress.WorkObservations))).To(Succeed())
	g.Expect(state.Root.Files).To(HaveLen(1))

	internalImports, ok := state.Root.Files[0].Quantity(metric.Name("internal.imports"))
	g.Expect(ok).To(BeTrue())
	g.Expect(internalImports).To(Equal(int64(1)))
}

//nolint:paralleltest // mutates the global provider registry
func TestHistoricalGoProviderReadsAttachedGitSource(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()
	runGitForSourceTest(t, dir, "init", "-b", "main")
	runGitForSourceTest(t, dir, "config", "user.name", "Test")
	runGitForSourceTest(t, dir, "config", "user.email", "test@example.com")
	g.Expect(os.WriteFile(
		filepath.Join(dir, "go.mod"),
		[]byte("module example.com/historical\n\ngo 1.26\n"),
		0o600,
	)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(dir, "main.go"),
		[]byte("package historical\n\nimport \"fmt\"\n\nfunc Value() string { return fmt.Sprint(1) }\n"),
		0o600,
	)).To(Succeed())
	runGitForSourceTest(t, dir, "add", ".")
	runGitForSourceTest(t, dir, "commit", "-m", "historical")
	oldCommit := runGitForSourceTest(t, dir, "rev-parse", "HEAD")
	g.Expect(os.WriteFile(filepath.Join(dir, "main.go"), []byte("package current\n"), 0o600)).To(Succeed())

	provider.ResetBaseRegistryForTesting()
	golang.Register()
	t.Cleanup(func() {
		provider.ResetBaseRegistryForTesting()
		filesystem.Register()
		git.Register()
	})

	state := &CommonState{
		TargetPath: dir,
		Flags:      &Flags{HistoryRange: git.HistoryRange{Until: "sha:" + oldCommit}},
		Requested:  RequestedMetrics{BaseMetrics: []metric.Name{golang.Imports}},
	}
	g.Expect(ResolveSource(state)).To(Succeed())
	g.Expect(ScanFilesystem(state, context.Background(), newTestSink(progress.WorkObservations))).To(Succeed())
	g.Expect(RunProviders(state, context.Background(), newTestSink(progress.WorkObservations))).To(Succeed())

	var mainFileFound bool

	for _, file := range state.Root.Files {
		if file.Name != "main.go" {
			continue
		}

		mainFileFound = true
		imports, ok := file.Quantity(golang.Imports)
		g.Expect(ok).To(BeTrue())
		g.Expect(imports).To(Equal(int64(1)))
	}

	g.Expect(mainFileFound).To(BeTrue())
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
