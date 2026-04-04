package scan

import (
	"os"
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

func TestScanPermissionDenied(t *testing.T) {
	g := NewGomegaWithT(t)

	// Create a temporary directory with an unreadable file
	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "readable.txt"))
	g.Expect(err).NotTo(HaveOccurred())
	f.WriteString("hello")
	f.Close()

	unreadable := filepath.Join(tmp, "unreadable.txt")
	err = os.WriteFile(unreadable, []byte("secret"), 0o000)
	g.Expect(err).NotTo(HaveOccurred())

	root, err := Scan(tmp)
	g.Expect(err).NotTo(HaveOccurred())
	// Scanner should continue and include the readable file
	g.Expect(len(root.Files)).To(BeNumerically(">=", 1))
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
