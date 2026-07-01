package stages

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// setupHistoryRepo creates a temp git repo with three commits touching two files.
// Commit 1 (Alice, 2024-01-01): adds a.go and b.go.
// Commit 2 (Bob,   2025-06-15): modifies b.go.
// Commit 3 (Alice, default):   adds c.go.
func setupHistoryRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	runAs := func(name, email string, args ...string) {
		t.Helper()

		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper
		cmd.Dir = dir

		cmd.Env = append(
			os.Environ(),
			"GIT_AUTHOR_NAME="+name, "GIT_AUTHOR_EMAIL="+email,
			"GIT_COMMITTER_NAME="+name, "GIT_COMMITTER_EMAIL="+email,
		)

		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("command %v failed: %s\n%s", args, err, out)
		}
	}

	runAs("Alice", "alice@example.com", "git", "init")
	runAs("Alice", "alice@example.com", "git", "config", "user.name", "Alice")
	runAs("Alice", "alice@example.com", "git", "config", "user.email", "alice@example.com")

	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "b.go"), []byte("package b\n"), 0o600)

	runAs("Alice", "alice@example.com", "git", "add", ".")
	runAs("Alice", "alice@example.com", "git", "commit", "-m", "initial", "--date=2024-01-01T00:00:00+00:00")

	_ = os.WriteFile(filepath.Join(dir, "b.go"), []byte("package b\n// edit\n"), 0o600)

	runAs("Bob", "bob@example.com", "git", "add", "b.go")
	runAs("Bob", "bob@example.com", "git", "commit", "-m", "bob edit", "--date=2025-06-15T00:00:00+00:00")

	_ = os.WriteFile(filepath.Join(dir, "c.go"), []byte("package c\n"), 0o600)

	runAs("Alice", "alice@example.com", "git", "add", "c.go")
	runAs("Alice", "alice@example.com", "git", "commit", "-m", "add c")

	return dir
}

func buildHistoryState(dir string) *CommonState {
	root := &model.Directory{Path: dir, Name: filepath.Base(dir)}

	for _, name := range []string{"a.go", "b.go", "c.go"} {
		root.Files = append(root.Files, &model.File{
			Path: filepath.Join(dir, name),
			Name: name,
		})
	}

	return &CommonState{
		TargetPath: dir,
		Root:       root,
		Flags:      &Flags{Config: &config.Config{}},
	}
}

func TestLoadGitHistory_PopulatesGitHistory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	state := buildHistoryState(setupHistoryRepo(t))

	g.Expect(LoadGitHistory(state)).To(Succeed())
	g.Expect(state.GitHistory).NotTo(BeEmpty())

	for _, c := range state.GitHistory {
		g.Expect(c.Hash).NotTo(BeEmpty())
		g.Expect(c.Author.When.IsZero()).To(BeFalse())
		g.Expect(c.ChangedPaths).NotTo(BeEmpty())
	}
}

func TestGroupGitHistoryByFile_PointsBackIntoGitHistory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	state := buildHistoryState(setupHistoryRepo(t))
	g.Expect(LoadGitHistory(state)).To(Succeed())
	g.Expect(GroupGitHistoryByFile(state)).To(Succeed())

	g.Expect(state.FileHistory).NotTo(BeEmpty())

	for _, refs := range state.FileHistory {
		for _, ref := range refs {
			g.Expect(ref.Commit).NotTo(BeNil())
			g.Expect(ref.When.IsZero()).To(BeFalse())
		}
	}
}

func TestGroupGitHistoryByFile_BFileTouchedByBothAuthors(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	state := buildHistoryState(setupHistoryRepo(t))
	g.Expect(LoadGitHistory(state)).To(Succeed())
	g.Expect(GroupGitHistoryByFile(state)).To(Succeed())

	var bFile *model.File

	for _, f := range state.Root.Files {
		if f.Name == "b.go" {
			bFile = f

			break
		}
	}

	g.Expect(bFile).NotTo(BeNil())
	refs := state.FileHistory[bFile]
	g.Expect(refs).To(HaveLen(2))

	authors := map[string]bool{}
	for _, r := range refs {
		authors[r.Commit.Author.Name] = true
	}

	g.Expect(authors).To(Equal(map[string]bool{"Alice": true, "Bob": true}))
}

func TestExtractFileHistory_ComputesMinMax(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	state := buildHistoryState(setupHistoryRepo(t))
	g.Expect(LoadGitHistory(state)).To(Succeed())
	g.Expect(GroupGitHistoryByFile(state)).To(Succeed())
	g.Expect(ExtractFileHistory(state)).To(Succeed())

	var bFile *model.File

	for _, f := range state.Root.Files {
		if f.Name == "b.go" {
			bFile = f

			break
		}
	}

	tr, ok := state.FileTimeRange[bFile]
	g.Expect(ok).To(BeTrue())

	expectedEarliest := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	expectedLatest := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	g.Expect(tr.Earliest.Equal(expectedEarliest)).To(BeTrue(), "got %v", tr.Earliest)
	g.Expect(tr.Latest.Equal(expectedLatest)).To(BeTrue(), "got %v", tr.Latest)
}

func TestCommitTimeRange_FoldsGlobalMinMax(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	state := buildHistoryState(setupHistoryRepo(t))
	g.Expect(LoadGitHistory(state)).To(Succeed())
	g.Expect(GroupGitHistoryByFile(state)).To(Succeed())
	g.Expect(ExtractFileHistory(state)).To(Succeed())

	global := CommitTimeRange(state.FileTimeRange)

	g.Expect(global.Earliest.IsZero()).To(BeFalse())
	g.Expect(global.Latest.After(global.Earliest) || global.Latest.Equal(global.Earliest)).To(BeTrue())
}

func TestCommitTimeRange_EmptyMap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tr := CommitTimeRange(nil)
	g.Expect(tr.Earliest.IsZero()).To(BeTrue())
	g.Expect(tr.Latest.IsZero()).To(BeTrue())
}

func TestPruneFileHistoryToTree_DropsFilesAbsentFromTree(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	kept := &model.File{Path: "root/a.go", Name: "a.go"}
	binary := &model.File{Path: "root/blob.bin", Name: "blob.bin", IsBinary: true}

	state := &CommonState{
		// Root simulates the tree after FilterBinaryFiles removed the binary file.
		Root: &model.Directory{Path: "root", Name: "root", Files: []*model.File{kept}},
		FileHistory: map[*model.File][]CommitRef{
			kept:   {{When: time.Unix(100, 0)}},
			binary: {{When: time.Unix(200, 0)}},
		},
		FileTimeRange: map[*model.File]TimeRange{
			kept:   {Earliest: time.Unix(100, 0), Latest: time.Unix(100, 0)},
			binary: {Earliest: time.Unix(200, 0), Latest: time.Unix(200, 0)},
		},
	}

	g.Expect(PruneFileHistoryToTree(state)).To(Succeed())

	g.Expect(state.FileHistory).To(HaveKey(kept))
	g.Expect(state.FileHistory).NotTo(HaveKey(binary))
	g.Expect(state.FileTimeRange).To(HaveKey(kept))
	g.Expect(state.FileTimeRange).NotTo(HaveKey(binary))

	// The pruned file must not widen the global commit time range.
	tr2 := CommitTimeRange(state.FileTimeRange)
	g.Expect(tr2.Latest).To(Equal(time.Unix(100, 0)))
}

func TestPruneFileHistoryToTree_NoHistoryIsNoOp(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	state := &CommonState{Root: &model.Directory{Path: "root", Name: "root"}}

	g.Expect(PruneFileHistoryToTree(state)).To(Succeed())
	g.Expect(state.FileHistory).To(BeEmpty())
	g.Expect(state.FileTimeRange).To(BeEmpty())
}
