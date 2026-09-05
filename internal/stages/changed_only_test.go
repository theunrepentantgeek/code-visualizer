package stages_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestFilterChangedOnly_DisabledIsNoOpWithoutGit(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	root := &model.Directory{
		Files: []*model.File{{Name: "one.go"}, {Name: "two.go"}},
	}
	state := &stages.CommonState{
		Root:  root,
		Flags: &stages.Flags{},
	}

	g.Expect(stages.FilterChangedOnly(state)).To(Succeed())
	g.Expect(root.Files).To(HaveLen(2))
}

func TestFilterChangedOnly_RequiresConstrainedRange(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	state := &stages.CommonState{
		Root:  &model.Directory{},
		Flags: &stages.Flags{ChangedOnly: true},
	}

	g.Expect(stages.FilterChangedOnly(state)).To(
		MatchError("--changed-only requires --from or --until"),
	)
}

func TestFilterChangedOnly_RequiresGitRepository(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := t.TempDir()
	state := &stages.CommonState{
		TargetPath: dir,
		Root:       &model.Directory{Path: dir},
		Flags: &stages.Flags{
			ChangedOnly:  true,
			HistoryRange: git.HistoryRange{From: "v1.0"},
		},
	}

	err := stages.FilterChangedOnly(state)

	var gitRequired *stages.GitRequiredError
	g.Expect(errors.As(err, &gitRequired)).To(BeTrue())
}

func TestFilterChangedOnly_PrunesUnchangedFilesAndEmptyDirectories(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := setupChangedOnlyRepo(t)
	root := &model.Directory{
		Path: dir,
		Name: filepath.Base(dir),
		Files: []*model.File{
			{Path: filepath.Join(dir, "changed.go"), Name: "changed.go"},
			{Path: filepath.Join(dir, "stable.go"), Name: "stable.go"},
		},
		Dirs: []*model.Directory{{
			Path: filepath.Join(dir, "nested"),
			Name: "nested",
			Files: []*model.File{{
				Path: filepath.Join(dir, "nested", "unchanged.go"),
				Name: "unchanged.go",
			}},
			DirectFileCount: 1,
			AllFileCount:    1,
		}},
		DirectFileCount: 2,
		AllFileCount:    3,
		AllDirCount:     1,
	}
	state := &stages.CommonState{
		TargetPath: dir,
		Root:       root,
		Flags: &stages.Flags{
			ChangedOnly:  true,
			HistoryRange: git.HistoryRange{From: "tag:before-change"},
		},
	}

	g.Expect(stages.FilterChangedOnly(state)).To(Succeed())
	g.Expect(root.Files).To(ConsistOf(HaveField("Name", "changed.go")))
	g.Expect(root.Dirs).To(BeEmpty())
	g.Expect(root.DirectFileCount).To(Equal(1))
	g.Expect(root.AllFileCount).To(Equal(1))
	g.Expect(root.AllDirCount).To(Equal(0))
}

func TestFilterChangedOnly_EmptyIntersectionReturnsTypedError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := setupChangedOnlyRepo(t)
	state := &stages.CommonState{
		TargetPath: dir,
		Root: &model.Directory{
			Path: dir,
			Files: []*model.File{{
				Path: filepath.Join(dir, "stable.go"),
				Name: "stable.go",
			}},
		},
		Flags: &stages.Flags{
			ChangedOnly:  true,
			HistoryRange: git.HistoryRange{From: "tag:before-change"},
		},
	}

	err := stages.FilterChangedOnly(state)

	var noFiles *stages.NoFilesAfterFilterError
	g.Expect(errors.As(err, &noFiles)).To(BeTrue())
	g.Expect(err).To(MatchError(
		"no files available for visualization after filtering to files changed in the selected Git range",
	))
}

func setupChangedOnlyRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()

		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // fixed test commands
		cmd.Dir = dir

		cmd.Env = append(
			os.Environ(),
			"GIT_AUTHOR_NAME=Alice",
			"GIT_AUTHOR_EMAIL=alice@example.com",
			"GIT_COMMITTER_NAME=Alice",
			"GIT_COMMITTER_EMAIL=alice@example.com",
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %s\n%s", args, err, output)
		}
	}

	g := NewGomegaWithT(t)
	g.Expect(os.MkdirAll(filepath.Join(dir, "nested"), 0o750)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "changed.go"), []byte("package changed\n"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "stable.go"), []byte("package stable\n"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(dir, "nested", "unchanged.go"),
		[]byte("package unchanged\n"),
		0o600,
	)).To(Succeed())

	run("git", "init", "-b", "main")
	run("git", "add", ".")
	run("git", "commit", "-m", "initial")
	run("git", "tag", "before-change")

	g.Expect(os.WriteFile(
		filepath.Join(dir, "changed.go"),
		[]byte("package changed\n// modified\n"),
		0o600,
	)).To(Succeed())
	run("git", "add", "changed.go")
	run("git", "commit", "-m", "modify changed file")

	return dir
}
