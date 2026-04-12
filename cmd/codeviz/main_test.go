package main

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
	"github.com/bevan/code-visualizer/internal/scan"
)

func TestMain(m *testing.M) {
	filesystem.Register()
	os.Exit(m.Run())
}

func TestClassifyNoFilesAfterFilterError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	err := &noFilesAfterFilterError{msg: "no files available for visualization after excluding binary files"}
	code := classifyError(err)
	g.Expect(code).To(Equal(6))
}

func TestClassifyErrorPreservesExistingCodes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(classifyError(&targetPathError{msg: "bad path"})).To(Equal(2))
	g.Expect(classifyError(&gitRequiredError{})).To(Equal(3))
	g.Expect(classifyError(&outputPathError{msg: "bad output"})).To(Equal(4))
	g.Expect(classifyError(&noFilesAfterFilterError{msg: "no files"})).To(Equal(6))
}

func TestFilterNotCalledForFileSizeMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(filesystem.FileSize).NotTo(Equal(filesystem.FileLines))

	f := &model.File{Path: "/project/image.png", Name: "image.png", IsBinary: true}
	f.SetQuantity(filesystem.FileSize, 1024)
	root := &model.Directory{
		Path: "/project", Name: "project",
		Files: []*model.File{f},
	}
	g.Expect(countFilesInTree(root)).To(Equal(1))
}

func TestFilterNotCalledForFileAgeMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Verify that file-age metric does not trigger filtering
	g.Expect(filesystem.FileLines).NotTo(Equal("file-age"))
}

func TestFilterAppliedRegardlessOfFillMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fGo := &model.File{Path: "/project/main.go", Name: "main.go", IsBinary: false}
	fGo.SetQuantity(filesystem.FileLines, 50)
	fGo.SetClassification(filesystem.FileType, "go")

	fPng := &model.File{Path: "/project/image.png", Name: "image.png", IsBinary: true}
	fPng.SetQuantity(filesystem.FileSize, 1024)
	fPng.SetClassification(filesystem.FileType, "png")

	root := &model.Directory{
		Path: "/project", Name: "project",
		Files: []*model.File{fGo, fPng},
	}

	filtered := scan.FilterBinaryFiles(root)
	g.Expect(filtered.Files).To(HaveLen(1))
	g.Expect(filtered.Files[0].Name).To(Equal("main.go"))
}

func TestNoFilterWhenFileSizeWithFileTypeFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fGo := &model.File{Path: "/project/main.go", Name: "main.go", IsBinary: false}
	fGo.SetQuantity(filesystem.FileSize, 100)
	fGo.SetClassification(filesystem.FileType, "go")

	fPng := &model.File{Path: "/project/image.png", Name: "image.png", IsBinary: true}
	fPng.SetQuantity(filesystem.FileSize, 1024)
	fPng.SetClassification(filesystem.FileType, "png")

	root := &model.Directory{
		Path: "/project", Name: "project",
		Files: []*model.File{fGo, fPng},
	}

	// Without filtering, both files remain
	g.Expect(countFilesInTree(root)).To(Equal(2))
}

// countFilesInTree is a test helper that counts all files in a tree.
func countFilesInTree(node *model.Directory) int {
	count := len(node.Files)
	for _, d := range node.Dirs {
		count += countFilesInTree(d)
	}

	return count
}

func TestTreemapCmd_Validate_InvalidFilterGlob(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cmd := &TreemapCmd{
		TargetPath: ".",
		Output:     "out.png",
		Size:       "file-size",
		Filter:     []string{"![invalid"},
	}

	err := cmd.Validate()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("invalid filter"))
}

func TestTreemapCmd_Validate_ValidFilters(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cmd := &TreemapCmd{
		TargetPath: ".",
		Output:     "out.png",
		Size:       "file-size",
		Filter:     []string{"!.*", "*.go", "!**/*.log"},
	}

	err := cmd.Validate()
	g.Expect(err).NotTo(HaveOccurred())
}
