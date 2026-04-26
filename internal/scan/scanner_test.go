package scan

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/filter"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
)

func TestScanFlat(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "flat")

	root, err := Scan(dir, nil, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).ToNot(BeNil())

	if root == nil {
		return
	}

	g.Expect(root.Name).To(Equal("flat"))
	g.Expect(root.Files).To(HaveLen(3))
	g.Expect(root.Dirs).To(BeEmpty())

	sizes := map[string]int64{}

	for _, f := range root.Files {
		v, ok := f.Quantity(filesystem.FileSize)
		g.Expect(ok).To(BeTrue())

		sizes[f.Name] = v
	}

	g.Expect(sizes["small.txt"]).To(Equal(int64(5)))
	g.Expect(sizes["medium.go"]).To(Equal(int64(100)))
	g.Expect(sizes["large.rs"]).To(Equal(int64(1000)))
}

func TestScanNested(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "nested")

	root, err := Scan(dir, nil, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).ToNot(BeNil())

	if root == nil {
		return
	}

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
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "empty")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	_, err := Scan(dir, nil, nil)
	g.Expect(err).To(MatchError(ContainSubstring("no files")))
}

func TestScanFollowsFileSymlinks(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "with-symlinks")

	root, err := Scan(dir, nil, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).ToNot(BeNil())

	if root == nil {
		return
	}

	fileNames := map[string]bool{}
	for _, f := range root.Files {
		fileNames[f.Name] = true
	}

	g.Expect(fileNames).To(HaveKey("real.txt"))
	g.Expect(fileNames).To(HaveKey("link-to-file.txt"))
}

func TestScanSkipsDirSymlinks(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "with-symlinks")

	root, err := Scan(dir, nil, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).ToNot(BeNil())

	if root == nil {
		return
	}

	dirNames := map[string]bool{}
	for _, d := range root.Dirs {
		dirNames[d.Name] = true
	}

	g.Expect(dirNames).To(HaveKey("target-dir"))
	g.Expect(dirNames).NotTo(HaveKey("link-to-dir"))
}

func TestScanFileExtension(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "flat")

	root, err := Scan(dir, nil, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).ToNot(BeNil())

	if root == nil {
		return
	}

	exts := map[string]string{}
	for _, f := range root.Files {
		exts[f.Name] = f.Extension
	}

	g.Expect(exts["small.txt"]).To(Equal("txt"))
	g.Expect(exts["medium.go"]).To(Equal("go"))
	g.Expect(exts["large.rs"]).To(Equal("rs"))
}

func TestScanSetsFileType(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "flat")

	root, err := Scan(dir, nil, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).ToNot(BeNil())

	if root == nil {
		return
	}

	for _, f := range root.Files {
		ft, ok := f.Classification(filesystem.FileType)
		g.Expect(ok).To(BeTrue())
		g.Expect(ft).NotTo(BeEmpty())
	}
}

func TestFilterBinaryFilesMixed(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Path: "/project",
		Name: "project",
		Files: []*model.File{
			{Path: "/project/main.go", Name: "main.go", IsBinary: false},
			{Path: "/project/image.png", Name: "image.png", IsBinary: true},
			{Path: "/project/util.go", Name: "util.go", IsBinary: false},
		},
	}

	filtered := FilterBinaryFiles(root)
	g.Expect(filtered.Files).To(HaveLen(2))
	g.Expect(filtered.Files[0].Name).To(Equal("main.go"))
	g.Expect(filtered.Files[1].Name).To(Equal("util.go"))
}

func TestFilterBinaryFilesPrunesEmptyDirs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Path: "/project",
		Name: "project",
		Files: []*model.File{
			{Path: "/project/main.go", Name: "main.go", IsBinary: false},
		},
		Dirs: []*model.Directory{
			{
				Path: "/project/assets",
				Name: "assets",
				Files: []*model.File{
					{Path: "/project/assets/logo.png", Name: "logo.png", IsBinary: true},
				},
			},
		},
	}

	filtered := FilterBinaryFiles(root)
	g.Expect(filtered.Files).To(HaveLen(1))
	g.Expect(filtered.Dirs).To(BeEmpty())
}

//nolint:paralleltest // mutates global slog default logger
func TestFilterBinaryFilesLogsExcluded(t *testing.T) {
	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	oldDefault := slog.Default()

	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(oldDefault)

	root := &model.Directory{
		Path: "/project",
		Name: "project",
		Files: []*model.File{
			{Path: "/project/main.go", Name: "main.go", IsBinary: false},
			{Path: "/project/image.png", Name: "image.png", IsBinary: true},
		},
	}

	_ = FilterBinaryFiles(root)

	g.Expect(buf.String()).To(ContainSubstring("excluding binary file"))
}

func TestScanWithRules_ExcludesDotfiles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "with-dotfiles")

	rules := []filter.Rule{
		{Pattern: ".*", Mode: filter.Exclude},
	}

	root, err := Scan(dir, rules, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).NotTo(BeNil())

	if root == nil {
		return
	}

	// .hidden and .config/ should be excluded
	// Only src/main.go and README.md should remain
	allFiles := collectFileNames(root)
	g.Expect(allFiles).To(ConsistOf("main.go", "README.md"))

	allDirs := collectDirNames(root)
	g.Expect(allDirs).To(ConsistOf("src"))
}

func TestScanWithRules_ExcludedDirNotDescended(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "with-dotfiles")

	rules := []filter.Rule{
		{Pattern: ".*", Mode: filter.Exclude},
	}

	root, err := Scan(dir, rules, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).NotTo(BeNil())

	if root == nil {
		return
	}

	// .config/ should not appear in the tree at all
	allDirs := collectDirNames(root)
	g.Expect(allDirs).NotTo(ContainElement(".config"))
}

func TestScanWithRules_NoRules_IncludesAll(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "with-dotfiles")

	root, err := Scan(dir, nil, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).NotTo(BeNil())

	if root == nil {
		return
	}

	allFiles := collectFileNames(root)
	g.Expect(allFiles).To(ContainElement("main.go"))
	g.Expect(allFiles).To(ContainElement("README.md"))
	g.Expect(allFiles).To(ContainElement(".hidden"))
	g.Expect(allFiles).To(ContainElement("settings.json"))
}

func TestScanWithRules_IncludeOverridesExclude(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "with-dotfiles")

	rules := []filter.Rule{
		{Pattern: ".config", Mode: filter.Include},
		{Pattern: ".config/**", Mode: filter.Include},
		{Pattern: ".*", Mode: filter.Exclude},
	}

	root, err := Scan(dir, rules, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).NotTo(BeNil())

	if root == nil {
		return
	}

	allFiles := collectFileNames(root)
	g.Expect(allFiles).To(ContainElement("settings.json"))
	g.Expect(allFiles).To(ContainElement("main.go"))
	g.Expect(allFiles).To(ContainElement("README.md"))
	g.Expect(allFiles).NotTo(ContainElement(".hidden"))
}

func TestScanWithRules_PrunesEmptyDirs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "with-dotfiles")

	// Exclude all .go and .json files — src/ and .config/ should be pruned
	rules := []filter.Rule{
		{Pattern: "**/*.go", Mode: filter.Exclude},
		{Pattern: "**/*.json", Mode: filter.Exclude},
	}

	root, err := Scan(dir, rules, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).NotTo(BeNil())

	if root == nil {
		return
	}

	allDirs := collectDirNames(root)
	g.Expect(allDirs).NotTo(ContainElement("src"))
	g.Expect(allDirs).NotTo(ContainElement(".config"))
}

func TestHasFiles_EmptyTree(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Path: "/empty", Name: "empty"}
	g.Expect(hasFiles(root)).To(BeFalse())
}

func TestHasFiles_FilesInRoot(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Path:  "/root",
		Name:  "root",
		Files: []*model.File{{Path: "/root/a.go", Name: "a.go"}},
	}
	g.Expect(hasFiles(root)).To(BeTrue())
}

func TestHasFiles_FilesOnlyInSubDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	child := &model.Directory{
		Path:  "/root/sub",
		Name:  "sub",
		Files: []*model.File{{Path: "/root/sub/b.go", Name: "b.go"}},
	}
	root := &model.Directory{Path: "/root", Name: "root", Dirs: []*model.Directory{child}}
	g.Expect(hasFiles(root)).To(BeTrue())
}

func TestHasFiles_EmptyDirsOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	child := &model.Directory{Path: "/root/sub", Name: "sub"}
	root := &model.Directory{Path: "/root", Name: "root", Dirs: []*model.Directory{child}}
	g.Expect(hasFiles(root)).To(BeFalse())
}

// collectFileNames collects all file names recursively.
func collectFileNames(dir *model.Directory) []string {
	names := make([]string, 0, len(dir.Files))
	for _, f := range dir.Files {
		names = append(names, f.Name)
	}

	for _, d := range dir.Dirs {
		names = append(names, collectFileNames(d)...)
	}

	return names
}

// collectDirNames collects all directory names recursively (excludes root).
func collectDirNames(dir *model.Directory) []string {
	names := make([]string, 0, len(dir.Dirs))
	for _, d := range dir.Dirs {
		names = append(names, d.Name)
		names = append(names, collectDirNames(d)...)
	}

	return names
}
