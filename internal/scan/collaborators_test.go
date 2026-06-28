package scan

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

type progressCall struct {
	path      string
	fileCount int
}

type recordingProgress struct {
	calls []progressCall
}

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

func (r *recordingProgress) OnDirectoryScanned(path string, fileCount int) {
	r.calls = append(r.calls, progressCall{path: path, fileCount: fileCount})
}

func TestFilterPolicyIncludesUsesRelativePath(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root, err := filepath.Abs(filepath.Join("testdata", "with-dotfiles"))
	g.Expect(err).NotTo(HaveOccurred())

	policy := newFilterPolicy(root, []filter.Rule{{Pattern: ".*", Mode: filter.Exclude}})

	included, relPath, err := policy.includes(filepath.Join(root, ".hidden"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(relPath).To(Equal(".hidden"))
	g.Expect(included).To(BeFalse())
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
	}, true)

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

func TestWalkerScanDirReportsProgressPerDirectory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root, err := filepath.Abs(filepath.Join("testdata", "nested"))
	g.Expect(err).NotTo(HaveOccurred())

	progress := &recordingProgress{}
	walker := newWalker(root, nil, progress, true)

	_, err = walker.scanDir(root)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(progress.calls).To(ConsistOf(
		progressCall{path: root, fileCount: 1},
		progressCall{path: filepath.Join(root, "sub"), fileCount: 1},
		progressCall{path: filepath.Join(root, "sub", "deep"), fileCount: 1},
	))
}
