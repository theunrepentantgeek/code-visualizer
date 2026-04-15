package folder

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/provider"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
	gitprovider "github.com/bevan/code-visualizer/internal/provider/git"
)

// setupGitRepo creates a temporary git repo with two files committed by different authors.
func setupGitRepo(t *testing.T) (dir string, sub string) {
	t.Helper()
	dir = t.TempDir()
	sub = filepath.Join(dir, "pkg")

	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	run := func(workdir string, env []string, args ...string) {
		t.Helper()

		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper runs known safe commands
		cmd.Dir = workdir

		cmd.Env = append(os.Environ(), env...)

		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %s\n%s", args, err, out)
		}
	}

	aliceEnv := []string{
		"GIT_AUTHOR_NAME=Alice", "GIT_AUTHOR_EMAIL=alice@example.com",
		"GIT_COMMITTER_NAME=Alice", "GIT_COMMITTER_EMAIL=alice@example.com",
	}
	bobEnv := []string{
		"GIT_AUTHOR_NAME=Bob", "GIT_AUTHOR_EMAIL=bob@example.com",
		"GIT_COMMITTER_NAME=Bob", "GIT_COMMITTER_EMAIL=bob@example.com",
	}

	run(dir, aliceEnv, "git", "init")
	run(dir, aliceEnv, "git", "config", "user.email", "alice@example.com")
	run(dir, aliceEnv, "git", "config", "user.name", "Alice")

	_ = os.WriteFile(filepath.Join(dir, "root.go"), []byte("package root\n"), 0o600)
	_ = os.WriteFile(filepath.Join(sub, "pkg.go"), []byte("package pkg\n"), 0o600)

	run(dir, aliceEnv, "git", "add", ".")
	run(dir, aliceEnv, "git", "commit", "-m", "initial", "--date=2023-01-01T00:00:00+00:00")

	_ = os.WriteFile(filepath.Join(sub, "pkg.go"), []byte("package pkg\n// updated\n"), 0o600)

	run(dir, bobEnv, "git", "add", "pkg/pkg.go")
	run(dir, bobEnv, "git", "commit", "-m", "bob update", "--date=2024-06-01T00:00:00+00:00")

	return dir, sub
}

// buildTreeWithSub builds a model tree: root with one file and one subdir with one file.
func buildTreeWithSub(dir, sub string) *model.Directory {
	rootFile := &model.File{Path: filepath.Join(dir, "root.go"), Name: "root.go"}
	subFile := &model.File{Path: filepath.Join(sub, "pkg.go"), Name: "pkg.go"}
	subDir := &model.Directory{
		Path:  sub,
		Name:  "pkg",
		Files: []*model.File{subFile},
	}

	return &model.Directory{
		Path:  dir,
		Name:  filepath.Base(dir),
		Files: []*model.File{rootFile},
		Dirs:  []*model.Directory{subDir},
	}
}

func TestFolderAuthorCountProviderMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := &FolderAuthorCountProvider{}
	g.Expect(p.Name()).To(Equal(FolderAuthorCount))
	g.Expect(p.Kind()).To(Equal(metric.Quantity))
	g.Expect(p.Scope()).To(Equal(provider.ScopeFolder))
	g.Expect(p.Description()).NotTo(BeEmpty())
	g.Expect(p.Dependencies()).To(BeEmpty())
}

func TestFolderAuthorCountProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir, sub := setupGitRepo(t)
	root := buildTreeWithSub(dir, sub)

	p := &FolderAuthorCountProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	// subdir pkg: only pkg.go which has 2 authors
	subDir := root.Dirs[0]
	count, ok := subDir.Quantity(FolderAuthorCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(count).To(Equal(int64(2)))

	// root: root.go (1 author) + pkg.go (2 authors) → 2 unique (Alice + Bob)
	rootCount, ok := root.Quantity(FolderAuthorCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(rootCount).To(Equal(int64(2)))
}

func TestFolderAuthorCountProviderNotGitRepo(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	root := &model.Directory{Path: dir, Name: "root"}

	p := &FolderAuthorCountProvider{}
	err := p.Load(root)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("git")))
}

func TestFolderAgeProviderMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := &FolderAgeProvider{}
	g.Expect(p.Name()).To(Equal(FolderAge))
	g.Expect(p.Kind()).To(Equal(metric.Quantity))
	g.Expect(p.Scope()).To(Equal(provider.ScopeFolder))
	g.Expect(p.Description()).NotTo(BeEmpty())
	g.Expect(p.Dependencies()).To(ConsistOf(gitprovider.FileAge))
}

func TestFolderAgeProvider(t *testing.T) { //nolint:dupl // similar structure is intentional — distinct metrics
	t.Parallel()
	g := NewGomegaWithT(t)

	rootFile := &model.File{Path: "/root/a.go", Name: "a.go"}
	subFile := &model.File{Path: "/root/pkg/b.go", Name: "b.go"}
	subDir := &model.Directory{Path: "/root/pkg", Name: "pkg", Files: []*model.File{subFile}}
	root := &model.Directory{
		Path:  "/root",
		Name:  "root",
		Files: []*model.File{rootFile},
		Dirs:  []*model.Directory{subDir},
	}

	rootFile.SetQuantity(gitprovider.FileAge, 100)
	subFile.SetQuantity(gitprovider.FileAge, 500)

	p := &FolderAgeProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	// subdir: max(500) = 500
	age, ok := subDir.Quantity(FolderAge)
	g.Expect(ok).To(BeTrue())
	g.Expect(age).To(Equal(int64(500)))

	// root: max(100, 500) = 500
	rootAge, ok := root.Quantity(FolderAge)
	g.Expect(ok).To(BeTrue())
	g.Expect(rootAge).To(Equal(int64(500)))
}

func TestFolderFreshnessProviderMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := &FolderFreshnessProvider{}
	g.Expect(p.Name()).To(Equal(FolderFreshness))
	g.Expect(p.Kind()).To(Equal(metric.Quantity))
	g.Expect(p.Scope()).To(Equal(provider.ScopeFolder))
	g.Expect(p.Description()).NotTo(BeEmpty())
	g.Expect(p.Dependencies()).To(ConsistOf(gitprovider.FileFreshness))
}

func TestFolderFreshnessProvider(t *testing.T) { //nolint:dupl // similar structure is intentional — distinct metrics
	t.Parallel()
	g := NewGomegaWithT(t)

	rootFile := &model.File{Path: "/root/a.go", Name: "a.go"}
	subFile := &model.File{Path: "/root/pkg/b.go", Name: "b.go"}
	subDir := &model.Directory{Path: "/root/pkg", Name: "pkg", Files: []*model.File{subFile}}
	root := &model.Directory{
		Path:  "/root",
		Name:  "root",
		Files: []*model.File{rootFile},
		Dirs:  []*model.Directory{subDir},
	}

	rootFile.SetQuantity(gitprovider.FileFreshness, 30)
	subFile.SetQuantity(gitprovider.FileFreshness, 5)

	p := &FolderFreshnessProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	// subdir: min(5) = 5
	fresh, ok := subDir.Quantity(FolderFreshness)
	g.Expect(ok).To(BeTrue())
	g.Expect(fresh).To(Equal(int64(5)))

	// root: min(30, 5) = 5
	rootFresh, ok := root.Quantity(FolderFreshness)
	g.Expect(ok).To(BeTrue())
	g.Expect(rootFresh).To(Equal(int64(5)))
}

func TestTotalFolderLinesProviderMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := &TotalFolderLinesProvider{}
	g.Expect(p.Name()).To(Equal(TotalFolderLines))
	g.Expect(p.Kind()).To(Equal(metric.Quantity))
	g.Expect(p.Scope()).To(Equal(provider.ScopeFolder))
	g.Expect(p.Description()).NotTo(BeEmpty())
	g.Expect(p.Dependencies()).To(ConsistOf(filesystem.FileLines))
}

func TestTotalFolderLinesProvider(t *testing.T) { //nolint:dupl // similar structure is intentional — distinct metrics
	t.Parallel()
	g := NewGomegaWithT(t)

	f1 := &model.File{Path: "/root/a.go", Name: "a.go"}
	f2 := &model.File{Path: "/root/pkg/b.go", Name: "b.go"}
	subDir := &model.Directory{Path: "/root/pkg", Name: "pkg", Files: []*model.File{f2}}
	root := &model.Directory{
		Path:  "/root",
		Name:  "root",
		Files: []*model.File{f1},
		Dirs:  []*model.Directory{subDir},
	}

	f1.SetQuantity(filesystem.FileLines, 10)
	f2.SetQuantity(filesystem.FileLines, 20)

	p := &TotalFolderLinesProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	subTotal, ok := subDir.Quantity(TotalFolderLines)
	g.Expect(ok).To(BeTrue())
	g.Expect(subTotal).To(Equal(int64(20)))

	rootTotal, ok := root.Quantity(TotalFolderLines)
	g.Expect(ok).To(BeTrue())
	g.Expect(rootTotal).To(Equal(int64(30)))
}

func TestTotalFolderSizeProviderMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := &TotalFolderSizeProvider{}
	g.Expect(p.Name()).To(Equal(TotalFolderSize))
	g.Expect(p.Kind()).To(Equal(metric.Quantity))
	g.Expect(p.Scope()).To(Equal(provider.ScopeFolder))
	g.Expect(p.Description()).NotTo(BeEmpty())
	g.Expect(p.Dependencies()).To(ConsistOf(filesystem.FileSize))
}

func TestTotalFolderSizeProvider(t *testing.T) { //nolint:dupl // similar structure is intentional — distinct metrics
	t.Parallel()
	g := NewGomegaWithT(t)

	f1 := &model.File{Path: "/root/a.go", Name: "a.go"}
	f2 := &model.File{Path: "/root/pkg/b.go", Name: "b.go"}
	subDir := &model.Directory{Path: "/root/pkg", Name: "pkg", Files: []*model.File{f2}}
	root := &model.Directory{
		Path:  "/root",
		Name:  "root",
		Files: []*model.File{f1},
		Dirs:  []*model.Directory{subDir},
	}

	f1.SetQuantity(filesystem.FileSize, 100)
	f2.SetQuantity(filesystem.FileSize, 200)

	p := &TotalFolderSizeProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	subSize, ok := subDir.Quantity(TotalFolderSize)
	g.Expect(ok).To(BeTrue())
	g.Expect(subSize).To(Equal(int64(200)))

	rootSize, ok := root.Quantity(TotalFolderSize)
	g.Expect(ok).To(BeTrue())
	g.Expect(rootSize).To(Equal(int64(300)))
}

func TestMeanFileAgeProviderMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := &MeanFileAgeProvider{}
	g.Expect(p.Name()).To(Equal(MeanFileAge))
	g.Expect(p.Kind()).To(Equal(metric.Measure))
	g.Expect(p.Scope()).To(Equal(provider.ScopeFolder))
	g.Expect(p.Description()).NotTo(BeEmpty())
	g.Expect(p.Dependencies()).To(ConsistOf(gitprovider.FileAge))
}

func TestMeanFileAgeProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f1 := &model.File{Path: "/root/a.go", Name: "a.go"}
	f2 := &model.File{Path: "/root/pkg/b.go", Name: "b.go"}
	f3 := &model.File{Path: "/root/pkg/c.go", Name: "c.go"}
	subDir := &model.Directory{Path: "/root/pkg", Name: "pkg", Files: []*model.File{f2, f3}}
	root := &model.Directory{
		Path:  "/root",
		Name:  "root",
		Files: []*model.File{f1},
		Dirs:  []*model.Directory{subDir},
	}

	f1.SetQuantity(gitprovider.FileAge, 100)
	f2.SetQuantity(gitprovider.FileAge, 200)
	f3.SetQuantity(gitprovider.FileAge, 300)

	p := &MeanFileAgeProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	// subdir mean: (200+300)/2 = 250
	subMean, ok := subDir.Measure(MeanFileAge)
	g.Expect(ok).To(BeTrue())
	g.Expect(subMean).To(BeNumerically("~", 250.0, 0.001))

	// root mean: (100+200+300)/3 = 200
	rootMean, ok := root.Measure(MeanFileAge)
	g.Expect(ok).To(BeTrue())
	g.Expect(rootMean).To(BeNumerically("~", 200.0, 0.001))
}

func TestMeanFileFreshnessProviderMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := &MeanFileFreshnessProvider{}
	g.Expect(p.Name()).To(Equal(MeanFileFreshness))
	g.Expect(p.Kind()).To(Equal(metric.Measure))
	g.Expect(p.Scope()).To(Equal(provider.ScopeFolder))
	g.Expect(p.Description()).NotTo(BeEmpty())
	g.Expect(p.Dependencies()).To(ConsistOf(gitprovider.FileFreshness))
}

func TestMeanFileFreshnessProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f1 := &model.File{Path: "/root/a.go", Name: "a.go"}
	f2 := &model.File{Path: "/root/b.go", Name: "b.go"}
	root := &model.Directory{
		Path:  "/root",
		Name:  "root",
		Files: []*model.File{f1, f2},
	}

	f1.SetQuantity(gitprovider.FileFreshness, 10)
	f2.SetQuantity(gitprovider.FileFreshness, 30)

	p := &MeanFileFreshnessProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	mean, ok := root.Measure(MeanFileFreshness)
	g.Expect(ok).To(BeTrue())
	g.Expect(mean).To(BeNumerically("~", 20.0, 0.001))
}

func TestMeanFileLinesProviderMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := &MeanFileLinesProvider{}
	g.Expect(p.Name()).To(Equal(MeanFileLines))
	g.Expect(p.Kind()).To(Equal(metric.Measure))
	g.Expect(p.Scope()).To(Equal(provider.ScopeFolder))
	g.Expect(p.Description()).NotTo(BeEmpty())
	g.Expect(p.Dependencies()).To(ConsistOf(filesystem.FileLines))
}

func TestMeanFileLinesProviderSkipsBinaryFiles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	text1 := &model.File{Path: "/root/a.go", Name: "a.go"}
	text2 := &model.File{Path: "/root/b.go", Name: "b.go"}
	binary := &model.File{Path: "/root/c.bin", Name: "c.bin", IsBinary: true}
	root := &model.Directory{
		Path:  "/root",
		Name:  "root",
		Files: []*model.File{text1, text2, binary},
	}

	text1.SetQuantity(filesystem.FileLines, 10)
	text2.SetQuantity(filesystem.FileLines, 20)
	// binary file: file-lines not set (as FileLinesProvider would leave it)

	p := &MeanFileLinesProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	// mean of text files only: (10+20)/2 = 15
	mean, ok := root.Measure(MeanFileLines)
	g.Expect(ok).To(BeTrue())
	g.Expect(mean).To(BeNumerically("~", 15.0, 0.001))
}

func TestMeanFileSizeProviderMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := &MeanFileSizeProvider{}
	g.Expect(p.Name()).To(Equal(MeanFileSize))
	g.Expect(p.Kind()).To(Equal(metric.Measure))
	g.Expect(p.Scope()).To(Equal(provider.ScopeFolder))
	g.Expect(p.Description()).NotTo(BeEmpty())
	g.Expect(p.Dependencies()).To(ConsistOf(filesystem.FileSize))
}

func TestMeanFileSizeProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f1 := &model.File{Path: "/root/a.go", Name: "a.go"}
	f2 := &model.File{Path: "/root/pkg/b.go", Name: "b.go"}
	subDir := &model.Directory{Path: "/root/pkg", Name: "pkg", Files: []*model.File{f2}}
	root := &model.Directory{
		Path:  "/root",
		Name:  "root",
		Files: []*model.File{f1},
		Dirs:  []*model.Directory{subDir},
	}

	f1.SetQuantity(filesystem.FileSize, 100)
	f2.SetQuantity(filesystem.FileSize, 300)

	p := &MeanFileSizeProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	subMean, ok := subDir.Measure(MeanFileSize)
	g.Expect(ok).To(BeTrue())
	g.Expect(subMean).To(BeNumerically("~", 300.0, 0.001))

	rootMean, ok := root.Measure(MeanFileSize)
	g.Expect(ok).To(BeTrue())
	g.Expect(rootMean).To(BeNumerically("~", 200.0, 0.001))
}

func TestFolderMetricsEmptyDirectory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Path: "/root", Name: "root"}

	for _, p := range []interface {
		Load(root *model.Directory) error
	}{
		&FolderAgeProvider{},
		&FolderFreshnessProvider{},
		&TotalFolderLinesProvider{},
		&TotalFolderSizeProvider{},
		&MeanFileAgeProvider{},
		&MeanFileFreshnessProvider{},
		&MeanFileLinesProvider{},
		&MeanFileSizeProvider{},
	} {
		err := p.Load(root)
		g.Expect(err).NotTo(HaveOccurred())
	}

	// No metrics should be set on an empty directory
	_, okAge := root.Quantity(FolderAge)
	g.Expect(okAge).To(BeFalse())

	_, okFresh := root.Quantity(FolderFreshness)
	g.Expect(okFresh).To(BeFalse())

	_, okLines := root.Quantity(TotalFolderLines)
	g.Expect(okLines).To(BeFalse())
}
