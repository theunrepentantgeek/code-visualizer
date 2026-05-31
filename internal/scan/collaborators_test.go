package scan

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

type stubDirEntry struct {
	name string
}

func (e stubDirEntry) Name() string {
	return e.name
}

func (stubDirEntry) IsDir() bool {
	return false
}

func (stubDirEntry) Type() os.FileMode {
	return 0
}

func (stubDirEntry) Info() (os.FileInfo, error) {
	return nil, os.ErrInvalid
}

func mustStatFile(t *testing.T, path string) os.FileInfo {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}

	if info == nil {
		t.Fatalf("stat %s returned nil info", path)
	}

	return info
}

func TestNodeBuilderProcessFileAddsMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := filepath.Join("testdata", "flat")
	entry := stubDirEntry{name: "small.txt"}
	entryPath := filepath.Join(dir, entry.Name())
	info := mustStatFile(t, entryPath)

	node := &model.Directory{Path: dir, Name: "flat"}
	builder := newNodeBuilder(func(path string) (bool, error) {
		g.Expect(path).To(Equal(entryPath))

		return true, nil
	})

	builder.processFile(node, entry, info, entryPath)

	g.Expect(node.Files).To(HaveLen(1))
	file := node.Files[0]
	g.Expect(file.Name).To(Equal("small.txt"))
	g.Expect(file.Path).To(Equal(entryPath))
	g.Expect(file.Extension).To(Equal("txt"))
	g.Expect(file.IsBinary).To(BeTrue())

	size, ok := file.Quantity(filesystem.FileSize)
	g.Expect(ok).To(BeTrue())
	g.Expect(size).To(Equal(int64(5)))

	fileType, ok := file.Classification(filesystem.FileType)
	g.Expect(ok).To(BeTrue())
	g.Expect(fileType).To(Equal("txt"))
}
