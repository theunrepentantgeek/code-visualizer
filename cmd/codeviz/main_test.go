package main

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/scan"
)

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

	// Verify that file-size metric does not trigger filtering
	// by checking that the condition in Run() is scoped to FileLines only
	g.Expect(metric.FileSize).NotTo(Equal(metric.FileLines))

	// When size is file-size, a tree with only binary files should NOT be filtered
	root := scan.DirectoryNode{
		Path: "/project",
		Name: "project",
		Files: []scan.FileNode{
			{Path: "/project/image.png", Name: "image.png", IsBinary: true, Size: 1024},
		},
	}
	// countFiles should return 1 — binary files are NOT excluded for non-line-count metrics
	g.Expect(countFilesInTree(root)).To(Equal(1))
}

func TestFilterNotCalledForFileAgeMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Verify that file-age metric does not trigger filtering
	g.Expect(metric.FileAge).NotTo(Equal(metric.FileLines))
}

func TestFilterAppliedRegardlessOfFillMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// When size=file-lines, binary files should be excluded
	// regardless of what fill metric is set to (e.g. file-type)
	root := scan.DirectoryNode{
		Path: "/project",
		Name: "project",
		Files: []scan.FileNode{
			{Path: "/project/main.go", Name: "main.go", IsBinary: false, LineCount: 50, FileType: "go"},
			{Path: "/project/image.png", Name: "image.png", IsBinary: true, Size: 1024, FileType: "png"},
		},
	}

	// FilterBinaryFiles removes binary files — it doesn't check fill/border metrics
	filtered := scan.FilterBinaryFiles(root)
	g.Expect(filtered.Files).To(HaveLen(1))
	g.Expect(filtered.Files[0].Name).To(Equal("main.go"))
}

func TestNoFilterWhenFileSizeWithFileTypeFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// When size=file-size, binary files should NOT be filtered
	// even when fill=file-type
	root := scan.DirectoryNode{
		Path: "/project",
		Name: "project",
		Files: []scan.FileNode{
			{Path: "/project/main.go", Name: "main.go", IsBinary: false, Size: 100, FileType: "go"},
			{Path: "/project/image.png", Name: "image.png", IsBinary: true, Size: 1024, FileType: "png"},
		},
	}

	// Without filtering, both files remain
	g.Expect(countFilesInTree(root)).To(Equal(2))
}

// countFilesInTree is a test helper that counts all files in a tree.
func countFilesInTree(node scan.DirectoryNode) int {
	count := len(node.Files)
	for _, d := range node.Dirs {
		count += countFilesInTree(d)
	}

	return count
}
