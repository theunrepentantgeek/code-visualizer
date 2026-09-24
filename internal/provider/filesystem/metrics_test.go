package filesystem

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

func TestFileLinesProviderReadsAttachedSource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	f := &model.File{
		Path:       "/virtual/main.go",
		SourcePath: "main.go",
		Source:     fstest.MapFS{"main.go": {Data: []byte("package main\n\nfunc main() {}\n")}},
	}
	root := &model.Directory{Files: []*model.File{f}}

	g.Expect((&FileLinesProvider{}).Load(root)).To(Succeed())

	lines, ok := f.Quantity(FileLines)
	g.Expect(ok).To(BeTrue())
	g.Expect(lines).To(Equal(int64(3)))
}

func TestFileSizeProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := FileSizeProvider{}
	root := &model.Directory{Path: "/root", Name: "root"}
	g.Expect(p.Load(root)).NotTo(HaveOccurred()) // no-op
}

func TestFileTypeProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := FileTypeProvider{}
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

	f := &model.File{Path: "bin.dat", Name: "bin.dat", IsBinary: true}
	root := &model.Directory{Name: "root", Files: []*model.File{f}}

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

func TestFileLinesProviderMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	RegisterBase()

	// Verify file-lines is registered as a base metric
	desc, ok := provider.GetBase(FileLines)
	g.Expect(ok).To(BeTrue())
	g.Expect(desc.Kind).To(Equal(metric.Quantity))
	g.Expect(desc.Description).NotTo(BeEmpty())
}

func TestFileLinesProviderCountsLongTextLine(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "long.txt")
	_ = os.WriteFile(path, make([]byte, 66000), 0o600)

	f := &model.File{Path: path, Name: "long.txt"}
	root := &model.Directory{Path: dir, Name: "root", Files: []*model.File{f}}

	p := FileLinesProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	lines, ok := f.Quantity(FileLines)
	g.Expect(ok).To(BeTrue())
	g.Expect(lines).To(Equal(int64(1)))
	g.Expect(f.IsBinary).To(BeFalse())
}

func TestFileLinesProviderCountsUTF16Lines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content []byte
	}{
		{
			name:    "little-endian",
			content: []byte{0xFF, 0xFE, 0x61, 0x00, 0x0A, 0x00, 0x62, 0x00, 0x0A, 0x00},
		},
		{
			name:    "big-endian",
			content: []byte{0xFE, 0xFF, 0x00, 0x61, 0x00, 0x0A, 0x00, 0x62, 0x00, 0x0A},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			dir := t.TempDir()
			_ = os.WriteFile(filepath.Join(dir, "code.cs"), tt.content, 0o600)

			f := &model.File{Path: filepath.Join(dir, "code.cs"), Name: "code.cs"}
			root := &model.Directory{Path: dir, Name: "root", Files: []*model.File{f}}

			p := FileLinesProvider{}
			err := p.Load(root)
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(f.IsBinary).To(BeFalse())

			v, ok := f.Quantity(FileLines)
			g.Expect(ok).To(BeTrue())
			g.Expect(v).To(Equal(int64(2)))
		})
	}
}

func TestFileLinesProviderHandlesEmptyFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "empty.txt"), []byte{}, 0o600)

	f := &model.File{Path: filepath.Join(dir, "empty.txt"), Name: "empty.txt"}
	root := &model.Directory{Path: dir, Name: "root", Files: []*model.File{f}}

	p := FileLinesProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(f.IsBinary).To(BeFalse())

	v, ok := f.Quantity(FileLines)
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(int64(0)))
}

func TestIsFilesystemMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(IsFilesystemMetric(FileSize)).To(BeTrue())
	g.Expect(IsFilesystemMetric(FileLines)).To(BeTrue())
	g.Expect(IsFilesystemMetric(FileType)).To(BeTrue())
	g.Expect(IsFilesystemMetric(metric.Name("commit-count"))).To(BeFalse())
}
