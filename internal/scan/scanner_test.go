package scan

import (
	"bytes"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/source"
)

type permissionFS struct {
	fstest.MapFS
}

func (p permissionFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == "blocked" {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrPermission}
	}

	entries, err := p.MapFS.ReadDir(name)
	if err != nil {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: err}
	}

	return entries, nil
}

func TestScanTreeReadsVirtualSource(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	tree := source.Tree{
		FS:       fstest.MapFS{"src/main.go": {Data: []byte("package main\n")}},
		RootName: "project",
		RootPath: "/display/project",
		RepoBase: "packages/project",
	}

	root, err := ScanTree(tree, nil, nil, true)
	g.Expect(err).NotTo(HaveOccurred())

	if root == nil {
		t.Fatal("expected scanned root")
	}

	g.Expect(root.Path).To(Equal("/display/project"))
	g.Expect(root.Files).To(BeEmpty())
	g.Expect(root.Dirs).To(HaveLen(1))
	g.Expect(root.Dirs[0].Files).To(HaveLen(1))
	g.Expect(root.Dirs[0].Files[0].Path).To(Equal("/display/project/src/main.go"))
	g.Expect(root.Dirs[0].Files[0].RepoPath).To(Equal("packages/project/src/main.go"))
	g.Expect(root.Dirs[0].Files[0].SourcePath).To(Equal("src/main.go"))
}

func TestScanTreeKeepsSymlinkIdentityWhileReadingTarget(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(dir, "target.txt"), []byte("target\n"), 0o600)).To(Succeed())
	g.Expect(os.Symlink("target.txt", filepath.Join(dir, "link.txt"))).To(Succeed())
	tree, err := source.WorkingTree(dir)
	g.Expect(err).NotTo(HaveOccurred())

	tree.RepoBase = "project"

	root, err := ScanTree(tree, nil, nil, true)
	g.Expect(err).NotTo(HaveOccurred())

	if root == nil {
		t.Fatal("expected scanned root")
	}

	var link *model.File

	for _, file := range root.Files {
		if file.Name == "link.txt" {
			link = file
		}
	}

	g.Expect(link).NotTo(BeNil())

	if link == nil {
		t.Fatal("expected scanned symlink")
	}

	g.Expect(link.RepoPath).To(Equal("project/link.txt"))
	g.Expect(link.SourcePath).To(Equal("target.txt"))
}

func TestScanTreeSkipsSymlinkChainEscapingSource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	parent := t.TempDir()
	dir := filepath.Join(parent, "source")
	g.Expect(os.Mkdir(dir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "safe.txt"), []byte("safe\n"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(parent, "outside.txt"), []byte("outside\n"), 0o600)).To(Succeed())
	g.Expect(os.Symlink("../outside.txt", filepath.Join(dir, "inside-link.txt"))).To(Succeed())
	g.Expect(os.Symlink("inside-link.txt", filepath.Join(dir, "chain-link.txt"))).To(Succeed())
	g.Expect(os.Symlink("..", filepath.Join(dir, "outside-dir"))).To(Succeed())
	g.Expect(os.Symlink("outside-dir/outside.txt", filepath.Join(dir, "component-link.txt"))).To(Succeed())

	tree, err := source.WorkingTree(dir)
	g.Expect(err).NotTo(HaveOccurred())

	root, err := ScanTree(tree, nil, nil, true)
	g.Expect(err).NotTo(HaveOccurred())

	if root == nil {
		t.Fatal("expected scanned root")
	}

	g.Expect(root.Files).To(HaveLen(1))
	g.Expect(root.Files[0].Name).To(Equal("safe.txt"))
}

func TestScanTreeFollowsAbsoluteSymlinkInsideSource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	g.Expect(os.WriteFile(target, []byte("target\n"), 0o600)).To(Succeed())
	g.Expect(os.Symlink(target, filepath.Join(dir, "link.txt"))).To(Succeed())

	tree, err := source.WorkingTree(dir)
	g.Expect(err).NotTo(HaveOccurred())

	root, err := ScanTree(tree, nil, nil, true)
	g.Expect(err).NotTo(HaveOccurred())

	if root == nil {
		t.Fatal("expected scanned root")
	}

	g.Expect(root.Files).To(HaveLen(2))
}

func TestScanTreeSkipsCyclicAndDeepSymlinks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fsys := fstest.MapFS{
		"safe.txt": {Data: []byte("safe\n")},
		"cycle-a":  {Data: []byte("cycle-b"), Mode: fs.ModeSymlink},
		"cycle-b":  {Data: []byte("cycle-a"), Mode: fs.ModeSymlink},
	}

	for depth := range maxSymlinkDepth + 1 {
		name := fmt.Sprintf("link-%02d", depth)

		target := "safe.txt"
		if depth < maxSymlinkDepth {
			target = fmt.Sprintf("link-%02d", depth+1)
		}

		fsys[name] = &fstest.MapFile{Data: []byte(target), Mode: fs.ModeSymlink}
	}

	root, err := ScanTree(source.Tree{FS: fsys, RootName: "root", RootPath: "/root"}, nil, nil, true)
	g.Expect(err).NotTo(HaveOccurred())

	if root == nil {
		t.Fatal("expected scanned root")
	}

	names := make(map[string]bool, len(root.Files))
	for _, file := range root.Files {
		names[file.Name] = true
	}

	g.Expect(names).NotTo(HaveKey("cycle-a"))
	g.Expect(names).NotTo(HaveKey("cycle-b"))
	g.Expect(names).NotTo(HaveKey("link-00"))
}

func TestScanTreeSkipsInaccessibleDirectories(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fsys := permissionFS{MapFS: fstest.MapFS{
		"safe.txt":         {Data: []byte("safe\n")},
		"blocked/file.txt": {Data: []byte("hidden\n")},
	}}

	root, err := ScanTree(source.Tree{FS: fsys, RootName: "root", RootPath: "/root"}, nil, nil, true)
	g.Expect(err).NotTo(HaveOccurred())

	if root == nil {
		t.Fatal("expected scanned root")
	}

	g.Expect(root.Files).To(HaveLen(1))
	g.Expect(root.Dirs).To(BeEmpty())
}

func TestScanFlat(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "flat")

	root, err := Scan(dir, nil, nil, true)
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

	root, err := Scan(dir, nil, nil, true)
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

	_, err := Scan(dir, nil, nil, true)
	g.Expect(err).To(MatchError(ContainSubstring("no files")))
}

func TestScanFollowsFileSymlinks(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "with-symlinks")

	root, err := Scan(dir, nil, nil, true)
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

	root, err := Scan(dir, nil, nil, true)
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

	root, err := Scan(dir, nil, nil, true)
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

	root, err := Scan(dir, nil, nil, true)
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

	root, err := Scan(dir, rules, nil, true)
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

	root, err := Scan(dir, rules, nil, true)
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

	root, err := Scan(dir, nil, nil, true)
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

	root, err := Scan(dir, rules, nil, true)
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

	root, err := Scan(dir, rules, nil, true)
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

func TestScanExcludesBinaryFiles_WhenIncludeBinaryFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()

	// Write a text file and a binary file (containing a null byte).
	g.Expect(os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "image.bin"), []byte("data\x00bytes"), 0o600)).To(Succeed())

	root, err := Scan(dir, nil, nil, false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).NotTo(BeNil())

	if root == nil {
		return
	}

	names := collectFileNames(root)
	g.Expect(names).To(ConsistOf("main.go"))
	g.Expect(names).NotTo(ContainElement("image.bin"))
}

func TestScanIncludesBinaryFiles_WhenIncludeBinaryTrue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()

	g.Expect(os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "image.bin"), []byte("data\x00bytes"), 0o600)).To(Succeed())

	root, err := Scan(dir, nil, nil, true)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).NotTo(BeNil())

	if root == nil {
		return
	}

	names := collectFileNames(root)
	g.Expect(names).To(ConsistOf("main.go", "image.bin"))

	// IsBinary flag is still populated.
	for _, f := range root.Files {
		if f.Name == "image.bin" {
			g.Expect(f.IsBinary).To(BeTrue())
		} else {
			g.Expect(f.IsBinary).To(BeFalse())
		}
	}
}

func TestScanFlat_FileCountsPopulated(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "flat")

	root, err := Scan(dir, nil, nil, true)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).ToNot(BeNil())

	if root == nil {
		return
	}

	// Flat directory: DirectFileCount == AllFileCount == number of files
	g.Expect(root.DirectFileCount).To(Equal(3), "DirectFileCount should match len(Files)")
	g.Expect(root.AllFileCount).To(Equal(3), "AllFileCount should equal DirectFileCount for a flat tree")
}

func TestScanNested_FileCountsPopulated(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "nested")

	root, err := Scan(dir, nil, nil, true)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root).ToNot(BeNil())

	if root == nil {
		return
	}

	// Tree: root(1 file) → sub(1 file) → deep(1 file)
	// root: DirectFileCount=1, AllFileCount=3
	g.Expect(root.DirectFileCount).To(Equal(1), "root DirectFileCount should be 1")
	g.Expect(root.AllFileCount).To(Equal(3), "root AllFileCount should include all descendants")

	sub := root.Dirs[0]
	g.Expect(sub.DirectFileCount).To(Equal(1), "sub DirectFileCount should be 1")
	g.Expect(sub.AllFileCount).To(Equal(2), "sub AllFileCount should include self and deep")

	deep := sub.Dirs[0]
	g.Expect(deep.DirectFileCount).To(Equal(1), "deep DirectFileCount should be 1")
	g.Expect(deep.AllFileCount).To(Equal(1), "deep AllFileCount should equal DirectFileCount for a leaf directory")
}

func TestFilterBinaryFiles_UpdatesFileCounts(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	child := &model.Directory{
		Path: "/project/sub",
		Name: "sub",
		Files: []*model.File{
			{Path: "/project/sub/util.go", Name: "util.go", IsBinary: false},
			{Path: "/project/sub/data.bin", Name: "data.bin", IsBinary: true},
		},
	}
	root := &model.Directory{
		Path: "/project",
		Name: "project",
		Files: []*model.File{
			{Path: "/project/main.go", Name: "main.go", IsBinary: false},
			{Path: "/project/image.png", Name: "image.png", IsBinary: true},
		},
		Dirs: []*model.Directory{child},
	}

	filtered := FilterBinaryFiles(root)

	// After filtering: root has 1 text file, sub has 1 text file
	g.Expect(filtered.DirectFileCount).To(Equal(1), "filtered root DirectFileCount should be 1")
	g.Expect(filtered.AllFileCount).To(Equal(2), "filtered root AllFileCount should include sub")

	g.Expect(filtered.Dirs).To(HaveLen(1))

	filteredSub := filtered.Dirs[0]
	g.Expect(filteredSub.DirectFileCount).To(Equal(1), "filtered sub DirectFileCount should be 1")
	g.Expect(filteredSub.AllFileCount).To(Equal(1), "filtered sub AllFileCount should equal DirectFileCount")
}

func TestFilterBinaryFiles_UpdatesDirCount(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	grandchild := &model.Directory{
		Path:  "/project/sub/deep",
		Name:  "deep",
		Files: []*model.File{{Path: "/project/sub/deep/util.go", Name: "util.go", IsBinary: false}},
	}
	child := &model.Directory{
		Path:  "/project/sub",
		Name:  "sub",
		Files: []*model.File{{Path: "/project/sub/main.go", Name: "main.go", IsBinary: false}},
		Dirs:  []*model.Directory{grandchild},
	}
	root := &model.Directory{
		Path:  "/project",
		Name:  "project",
		Files: []*model.File{{Path: "/project/root.go", Name: "root.go", IsBinary: false}},
		Dirs:  []*model.Directory{child},
	}

	filtered := FilterBinaryFiles(root)

	// root has child sub (AllDirCount = 1 direct + 1 grandchild = 2)
	g.Expect(filtered.AllDirCount).To(Equal(2), "filtered root AllDirCount should count sub and deep")
	// child has grandchild deep (AllDirCount = 1 direct)
	g.Expect(filtered.Dirs[0].AllDirCount).To(Equal(1), "filtered sub AllDirCount should count deep")
	// grandchild has no subdirs
	g.Expect(filtered.Dirs[0].Dirs[0].AllDirCount).To(Equal(0), "filtered deep AllDirCount should be 0")
}
