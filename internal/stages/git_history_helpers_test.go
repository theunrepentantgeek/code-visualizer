package stages

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// buildDirTree constructs a simple model.Directory tree rooted at repoRoot
// containing the provided file paths (absolute).
func buildDirTree(repoRoot string, paths []string) *model.Directory {
	root := &model.Directory{Path: repoRoot, Name: filepath.Base(repoRoot)}

	for _, p := range paths {
		root.Files = append(root.Files, &model.File{
			Path: p,
			Name: filepath.Base(p),
		})
	}

	return root
}

func TestWalkFilesWithRepoRelPaths_ReturnsSlashRelativePaths(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	repoRoot := "/repo"
	root := buildDirTree(repoRoot, []string{
		filepath.FromSlash("/repo/main.go"),
		filepath.FromSlash("/repo/pkg/util.go"),
	})

	got := map[string]bool{}

	walkFilesWithRepoRelPaths(root, repoRoot, func(rel string, _ *model.File) {
		got[rel] = true
	})

	g.Expect(got).To(HaveKey("main.go"))
	g.Expect(got).To(HaveKey("pkg/util.go"))
}

func TestWalkFilesWithRepoRelPaths_EmptyTree_CallsNothing(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Path: "/repo", Name: "repo"}
	calls := 0

	walkFilesWithRepoRelPaths(root, "/repo", func(_ string, _ *model.File) {
		calls++
	})

	g.Expect(calls).To(Equal(0))
}

func TestWalkFilesWithRepoRelPaths_CallbackReceivesCorrectFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	repoRoot := "/repo"
	file := &model.File{Path: filepath.FromSlash("/repo/foo.go"), Name: "foo.go"}
	root := &model.Directory{Path: repoRoot, Name: "repo", Files: []*model.File{file}}

	var gotFile *model.File

	walkFilesWithRepoRelPaths(root, repoRoot, func(_ string, f *model.File) {
		gotFile = f
	})

	g.Expect(gotFile).To(BeIdenticalTo(file))
}

func TestBuildTrackedPathSet_ContainsAllFiles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	repoRoot := "/repo"
	root := buildDirTree(repoRoot, []string{
		filepath.FromSlash("/repo/a.go"),
		filepath.FromSlash("/repo/b.go"),
	})

	tracked := buildTrackedPathSet(root, repoRoot)

	g.Expect(tracked).To(HaveKey("a.go"))
	g.Expect(tracked).To(HaveKey("b.go"))
	g.Expect(tracked).To(HaveLen(2))
}

func TestBuildTrackedPathSet_EmptyTree_ReturnsEmptyMap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Path: "/repo", Name: "repo"}

	tracked := buildTrackedPathSet(root, "/repo")

	g.Expect(tracked).To(BeEmpty())
}

func TestIndexFilesByRepoRelativePath_MapsRelPathToFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	repoRoot := "/repo"
	fileA := &model.File{Path: filepath.FromSlash("/repo/a.go"), Name: "a.go"}
	fileB := &model.File{Path: filepath.FromSlash("/repo/sub/b.go"), Name: "b.go"}
	root := &model.Directory{
		Path:  repoRoot,
		Name:  "repo",
		Files: []*model.File{fileA},
		Dirs: []*model.Directory{
			{
				Path:  filepath.FromSlash("/repo/sub"),
				Name:  "sub",
				Files: []*model.File{fileB},
			},
		},
	}

	index := indexFilesByRepoRelativePath(root, repoRoot)

	g.Expect(index).To(HaveKeyWithValue("a.go", fileA))
	g.Expect(index).To(HaveKeyWithValue("sub/b.go", fileB))
	g.Expect(index).To(HaveLen(2))
}

func TestIndexFilesByRepoRelativePath_EmptyTree_ReturnsEmptyMap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Path: "/repo", Name: "repo"}

	index := indexFilesByRepoRelativePath(root, "/repo")

	g.Expect(index).To(BeEmpty())
}
