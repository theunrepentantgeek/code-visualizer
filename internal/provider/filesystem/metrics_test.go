package filesystem

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/provider"
)

func TestFileSizeProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := FileSizeProvider{}
	g.Expect(p.Name()).To(Equal(FileSize))
	g.Expect(p.Kind()).To(Equal(metric.Quantity))
	g.Expect(p.Scope()).To(Equal(provider.ScopeFile))
	g.Expect(p.Description()).NotTo(BeEmpty())
	g.Expect(p.Dependencies()).To(BeNil())

	root := &model.Directory{Path: "/root", Name: "root"}
	g.Expect(p.Load(root)).NotTo(HaveOccurred()) // no-op
}

func TestFileTypeProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := FileTypeProvider{}
	g.Expect(p.Name()).To(Equal(FileType))
	g.Expect(p.Kind()).To(Equal(metric.Classification))
	g.Expect(p.Scope()).To(Equal(provider.ScopeFile))
	g.Expect(p.Description()).NotTo(BeEmpty())
	g.Expect(p.Dependencies()).To(BeNil())

	root := &model.Directory{Path: "/root", Name: "root"}
	g.Expect(p.Load(root)).NotTo(HaveOccurred()) // no-op
}

func TestFileLinesProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "three.go"), []byte("a\nb\nc\n"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "one.txt"), []byte("single\n"), 0o600)

	f1 := &model.File{Path: filepath.Join(dir, "three.go"), Name: "three.go", Extension: "go"}
	f2 := &model.File{Path: filepath.Join(dir, "one.txt"), Name: "one.txt", Extension: "txt"}
	root := &model.Directory{
		Path:  dir,
		Name:  "root",
		Files: []*model.File{f1, f2},
	}

	p := FileLinesProvider{}
	g.Expect(p.Scope()).To(Equal(provider.ScopeFile))
	g.Expect(p.Description()).NotTo(BeEmpty())
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	v1, ok := f1.Quantity(FileLines)
	g.Expect(ok).To(BeTrue())
	g.Expect(v1).To(Equal(int64(3)))

	v2, ok := f2.Quantity(FileLines)
	g.Expect(ok).To(BeTrue())
	g.Expect(v2).To(Equal(int64(1)))
}

func TestFileLinesProviderSkipsBinaryFiles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	// Write a single line longer than bufio.MaxScanTokenSize (65536) to trigger binary detection
	_ = os.WriteFile(filepath.Join(dir, "bin.dat"), append([]byte("hello\x00world"), make([]byte, 66000)...), 0o600)

	f := &model.File{Path: filepath.Join(dir, "bin.dat"), Name: "bin.dat"}
	root := &model.Directory{Path: dir, Name: "root", Files: []*model.File{f}}

	p := FileLinesProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	_, ok := f.Quantity(FileLines)
	g.Expect(ok).To(BeFalse())
	g.Expect(f.IsBinary).To(BeTrue())
}

func TestFileLinesProviderNestedDirs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	_ = os.MkdirAll(sub, 0o755)
	_ = os.WriteFile(filepath.Join(sub, "deep.go"), []byte("a\nb\n"), 0o600)

	f := &model.File{Path: filepath.Join(sub, "deep.go"), Name: "deep.go", Extension: "go"}
	root := &model.Directory{
		Path: dir,
		Name: "root",
		Dirs: []*model.Directory{
			{Path: sub, Name: "sub", Files: []*model.File{f}},
		},
	}

	p := FileLinesProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	v, ok := f.Quantity(FileLines)
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(int64(2)))
}
