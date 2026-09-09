package source_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/source"
)

func TestWorkingTreeReadsThroughModelFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(dir, "src"), 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "src", "main.go"), []byte("package main\n"), 0o600)).To(Succeed())

	tree, err := source.WorkingTree(dir)
	g.Expect(err).NotTo(HaveOccurred())

	file := model.File{
		Path:     "src/main.go",
		RepoPath: "project/src/main.go",
		Source:   tree.FS,
	}

	data, err := file.ReadAll()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data).To(Equal([]byte("package main\n")))
	g.Expect(tree.RepoPath("src/main.go")).To(Equal("src/main.go"))
}
