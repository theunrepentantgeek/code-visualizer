package scan

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

func TestScanFlat(t *testing.T) {
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "flat")

	root, err := Scan(dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root.Name).To(Equal("flat"))
	g.Expect(root.Files).To(HaveLen(3))
	g.Expect(root.Dirs).To(BeEmpty())

	sizes := map[string]int64{}
	for _, f := range root.Files {
		sizes[f.Name] = f.Size
	}
	g.Expect(sizes["small.txt"]).To(Equal(int64(5)))
	g.Expect(sizes["medium.go"]).To(Equal(int64(100)))
	g.Expect(sizes["large.rs"]).To(Equal(int64(1000)))
}

func TestScanNested(t *testing.T) {
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "nested")

	root, err := Scan(dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root.Name).To(Equal("nested"))
	g.Expect(root.Files).To(HaveLen(1))
	g.Expect(root.Dirs).To(HaveLen(1))

	sub := root.Dirs[0]
	g.Expect(sub.Name).To(Equal("sub"))
	g.Expect(sub.Files).To(HaveLen(1))
	g.Expect(sub.Dirs).To(HaveLen(1))

	deep := sub.Dirs[0]
	g.Expect(deep.Name).To(Equal("deep"))
	g.Expect(deep.Files).To(HaveLen(1))
	g.Expect(deep.Files[0].Name).To(Equal("leaf.md"))
}

func TestScanEmptyDir(t *testing.T) {
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "empty")

	_, err := Scan(dir)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("no files"))
}

func TestScanFollowsFileSymlinks(t *testing.T) {
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "with-symlinks")

	root, err := Scan(dir)
	g.Expect(err).NotTo(HaveOccurred())

	fileNames := map[string]bool{}
	for _, f := range root.Files {
		fileNames[f.Name] = true
	}
	g.Expect(fileNames).To(HaveKey("real.txt"))
	g.Expect(fileNames).To(HaveKey("link-to-file.txt"))
}

func TestScanSkipsDirSymlinks(t *testing.T) {
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "with-symlinks")

	root, err := Scan(dir)
	g.Expect(err).NotTo(HaveOccurred())

	dirNames := map[string]bool{}
	for _, d := range root.Dirs {
		dirNames[d.Name] = true
	}
	// The real target-dir should be present but the symlink link-to-dir should be skipped
	g.Expect(dirNames).To(HaveKey("target-dir"))
	g.Expect(dirNames).NotTo(HaveKey("link-to-dir"))
}

func TestScanFileExtension(t *testing.T) {
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "flat")

	root, err := Scan(dir)
	g.Expect(err).NotTo(HaveOccurred())

	exts := map[string]string{}
	for _, f := range root.Files {
		exts[f.Name] = f.Extension
	}
	g.Expect(exts["small.txt"]).To(Equal("txt"))
	g.Expect(exts["medium.go"]).To(Equal("go"))
	g.Expect(exts["large.rs"]).To(Equal("rs"))
}

func TestFilterBinaryFilesMixed(t *testing.T) {
	g := NewGomegaWithT(t)

	root := DirectoryNode{
		Path: "/project",
		Name: "project",
		Files: []FileNode{
			{Path: "/project/main.go", Name: "main.go", IsBinary: false, LineCount: 50},
			{Path: "/project/image.png", Name: "image.png", IsBinary: true, LineCount: 0},
			{Path: "/project/util.go", Name: "util.go", IsBinary: false, LineCount: 30},
		},
	}

	filtered := FilterBinaryFiles(root)
	g.Expect(filtered.Files).To(HaveLen(2))
	g.Expect(filtered.Files[0].Name).To(Equal("main.go"))
	g.Expect(filtered.Files[1].Name).To(Equal("util.go"))
}

func TestFilterBinaryFilesAllBinary(t *testing.T) {
	g := NewGomegaWithT(t)

	root := DirectoryNode{
		Path: "/project",
		Name: "project",
		Files: []FileNode{
			{Path: "/project/image.png", Name: "image.png", IsBinary: true},
			{Path: "/project/font.ttf", Name: "font.ttf", IsBinary: true},
		},
	}

	filtered := FilterBinaryFiles(root)
	g.Expect(filtered.Files).To(BeEmpty())
	g.Expect(countFiles(filtered)).To(Equal(0))
}

func TestFilterBinaryFilesNoBinary(t *testing.T) {
	g := NewGomegaWithT(t)

	root := DirectoryNode{
		Path: "/project",
		Name: "project",
		Files: []FileNode{
			{Path: "/project/main.go", Name: "main.go", IsBinary: false, LineCount: 50},
			{Path: "/project/README.md", Name: "README.md", IsBinary: false, LineCount: 10},
		},
	}

	filtered := FilterBinaryFiles(root)
	g.Expect(filtered.Files).To(HaveLen(2))
	g.Expect(filtered.Files[0].Name).To(Equal("main.go"))
	g.Expect(filtered.Files[1].Name).To(Equal("README.md"))
}

func TestFilterBinaryFilesPrunesEmptyDirs(t *testing.T) {
	g := NewGomegaWithT(t)

	root := DirectoryNode{
		Path: "/project",
		Name: "project",
		Files: []FileNode{
			{Path: "/project/main.go", Name: "main.go", IsBinary: false, LineCount: 50},
		},
		Dirs: []DirectoryNode{
			{
				Path: "/project/assets",
				Name: "assets",
				Files: []FileNode{
					{Path: "/project/assets/logo.png", Name: "logo.png", IsBinary: true},
					{Path: "/project/assets/icon.ico", Name: "icon.ico", IsBinary: true},
				},
			},
		},
	}

	filtered := FilterBinaryFiles(root)
	g.Expect(filtered.Files).To(HaveLen(1))
	g.Expect(filtered.Dirs).To(BeEmpty())
}

func TestFilterBinaryFilesNestedPruning(t *testing.T) {
	g := NewGomegaWithT(t)

	root := DirectoryNode{
		Path: "/project",
		Name: "project",
		Files: []FileNode{
			{Path: "/project/main.go", Name: "main.go", IsBinary: false, LineCount: 50},
		},
		Dirs: []DirectoryNode{
			{
				Path: "/project/src",
				Name: "src",
				Files: []FileNode{
					{Path: "/project/src/util.go", Name: "util.go", IsBinary: false, LineCount: 20},
				},
				Dirs: []DirectoryNode{
					{
						Path: "/project/src/bin",
						Name: "bin",
						Files: []FileNode{
							{Path: "/project/src/bin/app.exe", Name: "app.exe", IsBinary: true},
						},
					},
				},
			},
		},
	}

	filtered := FilterBinaryFiles(root)
	g.Expect(filtered.Files).To(HaveLen(1))
	g.Expect(filtered.Dirs).To(HaveLen(1))
	g.Expect(filtered.Dirs[0].Name).To(Equal("src"))
	g.Expect(filtered.Dirs[0].Files).To(HaveLen(1))
	g.Expect(filtered.Dirs[0].Dirs).To(BeEmpty()) // bin dir pruned
}

func TestFilterBinaryFilesLogsExcluded(t *testing.T) {
	g := NewGomegaWithT(t)

	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	oldDefault := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(oldDefault)

	root := DirectoryNode{
		Path: "/project",
		Name: "project",
		Files: []FileNode{
			{Path: "/project/main.go", Name: "main.go", IsBinary: false},
			{Path: "/project/image.png", Name: "image.png", IsBinary: true},
		},
	}

	_ = FilterBinaryFiles(root)
	g.Expect(buf.String()).To(ContainSubstring("excluding binary file"))
	g.Expect(buf.String()).To(ContainSubstring("image.png"))
}
