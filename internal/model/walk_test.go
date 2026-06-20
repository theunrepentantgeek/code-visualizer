package model

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestWalkFiles_FlatDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &Directory{
		Path:  "/root",
		Name:  "root",
		Files: []*File{{Path: "/root/a.go", Name: "a.go"}, {Path: "/root/b.go", Name: "b.go"}},
	}

	var names []string

	WalkFiles(root, func(f *File) { names = append(names, f.Name) })

	g.Expect(names).To(ConsistOf("a.go", "b.go"))
}

func TestWalkFiles_Nested(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	child := &Directory{
		Path:  "/root/sub",
		Name:  "sub",
		Files: []*File{{Path: "/root/sub/c.go", Name: "c.go"}},
	}
	root := &Directory{
		Path:  "/root",
		Name:  "root",
		Files: []*File{{Path: "/root/a.go", Name: "a.go"}},
		Dirs:  []*Directory{child},
	}

	var names []string

	WalkFiles(root, func(f *File) { names = append(names, f.Name) })

	g.Expect(names).To(ConsistOf("a.go", "c.go"))
}

func TestWalkFiles_Empty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &Directory{Path: "/root", Name: "root"}

	var count int

	WalkFiles(root, func(_ *File) { count++ })

	g.Expect(count).To(Equal(0))
}

func TestWalkFiles_DeepNesting(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	deep := &Directory{
		Path:  "/root/a/b",
		Name:  "b",
		Files: []*File{{Path: "/root/a/b/leaf.go", Name: "leaf.go"}},
	}
	mid := &Directory{
		Path:  "/root/a",
		Name:  "a",
		Files: []*File{{Path: "/root/a/mid.go", Name: "mid.go"}},
		Dirs:  []*Directory{deep},
	}
	root := &Directory{
		Path:  "/root",
		Name:  "root",
		Files: []*File{{Path: "/root/root.go", Name: "root.go"}},
		Dirs:  []*Directory{mid},
	}

	var names []string

	WalkFiles(root, func(f *File) { names = append(names, f.Name) })

	g.Expect(names).To(ConsistOf("root.go", "mid.go", "leaf.go"))
}

func TestWalkDirectories_FlatDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &Directory{Path: "/root", Name: "root"}

	var names []string

	WalkDirectories(root, func(d *Directory) { names = append(names, d.Name) })

	g.Expect(names).To(Equal([]string{"root"}))
}

func TestWalkDirectories_IncludesRoot(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	child := &Directory{Path: "/root/sub", Name: "sub"}
	root := &Directory{Path: "/root", Name: "root", Dirs: []*Directory{child}}

	var names []string

	WalkDirectories(root, func(d *Directory) { names = append(names, d.Name) })

	g.Expect(names).To(ConsistOf("root", "sub"))
}

func TestWalkDirectories_PostOrder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Build a two-level tree: root → [a, b]
	a := &Directory{Path: "/root/a", Name: "a"}
	b := &Directory{Path: "/root/b", Name: "b"}
	root := &Directory{Path: "/root", Name: "root", Dirs: []*Directory{a, b}}

	var names []string

	WalkDirectories(root, func(d *Directory) { names = append(names, d.Name) })

	// Post-order: children before parent
	g.Expect(names).To(HaveLen(3))

	if names != nil { // Keeping nilaway happy
		g.Expect(names[len(names)-1]).To(Equal("root"), "root must be visited last (post-order)")
	}
}

func TestWalkDirectories_DeepNesting_PostOrder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	leaf := &Directory{Path: "/root/a/b", Name: "b"}
	mid := &Directory{Path: "/root/a", Name: "a", Dirs: []*Directory{leaf}}
	root := &Directory{Path: "/root", Name: "root", Dirs: []*Directory{mid}}

	var names []string

	WalkDirectories(root, func(d *Directory) { names = append(names, d.Name) })

	g.Expect(names).To(Equal([]string{"b", "a", "root"}))
}

func TestWalkDirectories_Empty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &Directory{Path: "/root", Name: "root"}

	var count int

	WalkDirectories(root, func(_ *Directory) { count++ })

	g.Expect(count).To(Equal(1)) // only root
}

func TestCountFiles_Empty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &Directory{Path: "/root", Name: "root"}

	g.Expect(CountFiles(root)).To(Equal(0))
}

func TestCountFiles_FlatDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &Directory{
		Path:  "/root",
		Name:  "root",
		Files: []*File{{Name: "a.go"}, {Name: "b.go"}},
	}

	g.Expect(CountFiles(root)).To(Equal(2))
}

func TestCountFiles_Nested(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	child := &Directory{
		Path:  "/root/sub",
		Name:  "sub",
		Files: []*File{{Name: "c.go"}},
	}
	root := &Directory{
		Path:  "/root",
		Name:  "root",
		Files: []*File{{Name: "a.go"}},
		Dirs:  []*Directory{child},
	}

	g.Expect(CountFiles(root)).To(Equal(2))
}

func TestCountDirs_Empty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &Directory{Path: "/root", Name: "root"}

	g.Expect(CountDirs(root)).To(Equal(0))
}

func TestCountDirs_FlatDirs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := &Directory{Path: "/root/a", Name: "a"}
	b := &Directory{Path: "/root/b", Name: "b"}
	root := &Directory{Path: "/root", Name: "root", Dirs: []*Directory{a, b}}

	// root itself is not counted
	g.Expect(CountDirs(root)).To(Equal(2))
}

func TestCountDirs_Nested(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	leaf := &Directory{Path: "/root/a/b", Name: "b"}
	mid := &Directory{Path: "/root/a", Name: "a", Dirs: []*Directory{leaf}}
	root := &Directory{Path: "/root", Name: "root", Dirs: []*Directory{mid}}

	g.Expect(CountDirs(root)).To(Equal(2))
}

func TestCountDeclarations_Empty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &Directory{Path: "/root", Name: "root"}

	g.Expect(CountDeclarations(root)).To(Equal(0))
}

func TestCountDeclarations_FlatDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	decl1 := &Declaration{Name: "Foo", Kind: "function"}
	decl2 := &Declaration{Name: "Bar", Kind: "function"}
	root := &Directory{
		Path:  "/root",
		Name:  "root",
		Files: []*File{{Name: "a.go", Declarations: []*Declaration{decl1, decl2}}},
	}

	g.Expect(CountDeclarations(root)).To(Equal(2))
}

func TestCountDeclarations_Nested(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	decl1 := &Declaration{Name: "Foo", Kind: "function"}
	decl2 := &Declaration{Name: "Bar", Kind: "struct"}
	decl3 := &Declaration{Name: "Baz", Kind: "method"}
	child := &Directory{
		Path:  "/root/sub",
		Name:  "sub",
		Files: []*File{{Name: "b.go", Declarations: []*Declaration{decl2, decl3}}},
	}
	root := &Directory{
		Path:  "/root",
		Name:  "root",
		Files: []*File{{Name: "a.go", Declarations: []*Declaration{decl1}}},
		Dirs:  []*Directory{child},
	}

	g.Expect(CountDeclarations(root)).To(Equal(3))
}

func TestCountCommits_Empty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &Directory{Path: "/root", Name: "root"}

	g.Expect(CountCommits(root)).To(Equal(0))
}

func TestCountCommits_FlatDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c1 := &Commit{Hash: "aaa"}
	c2 := &Commit{Hash: "bbb"}
	root := &Directory{
		Path:  "/root",
		Name:  "root",
		Files: []*File{{Name: "a.go", Commits: []*Commit{c1, c2}}},
	}

	g.Expect(CountCommits(root)).To(Equal(2))
}

func TestCountCommits_Nested(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c1 := &Commit{Hash: "aaa"}
	c2 := &Commit{Hash: "bbb"}
	c3 := &Commit{Hash: "ccc"}
	child := &Directory{
		Path:  "/root/sub",
		Name:  "sub",
		Files: []*File{{Name: "b.go", Commits: []*Commit{c2, c3}}},
	}
	root := &Directory{
		Path:  "/root",
		Name:  "root",
		Files: []*File{{Name: "a.go", Commits: []*Commit{c1}}},
		Dirs:  []*Directory{child},
	}

	g.Expect(CountCommits(root)).To(Equal(3))
}

func TestWalkDeclarations_CollectsAllDescendants(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	decl1 := &Declaration{Name: "Foo", Kind: "function", Visibility: "public"}
	decl2 := &Declaration{Name: "bar", Kind: "method", Visibility: "private"}
	decl3 := &Declaration{Name: "Baz", Kind: "struct", Visibility: "public"}

	root := &Directory{
		Name: "root",
		Files: []*File{
			{Name: "a.go", Declarations: []*Declaration{decl1}},
		},
		Dirs: []*Directory{
			{
				Name: "sub",
				Files: []*File{
					{Name: "b.go", Declarations: []*Declaration{decl2, decl3}},
				},
			},
		},
	}

	var names []string

	WalkDeclarations(root, func(d *Declaration, _ *File) {
		names = append(names, d.Name)
	})

	g.Expect(names).To(ConsistOf("Foo", "bar", "Baz"))
}

func TestWalkDeclarations_Empty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &Directory{
		Name:  "root",
		Files: []*File{{Name: "a.go"}},
	}

	var count int

	WalkDeclarations(root, func(_ *Declaration, _ *File) { count++ })

	g.Expect(count).To(Equal(0))
}

func TestWalkCommits_CollectsAllDescendants(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c1 := &Commit{Hash: "aaa"}
	c2 := &Commit{Hash: "bbb"}

	root := &Directory{
		Name: "root",
		Files: []*File{
			{Name: "a.go", Commits: []*Commit{c1}},
		},
		Dirs: []*Directory{
			{
				Name: "sub",
				Files: []*File{
					{Name: "b.go", Commits: []*Commit{c2}},
				},
			},
		},
	}

	var hashes []string

	WalkCommits(root, func(c *Commit, _ *File) {
		hashes = append(hashes, c.Hash)
	})

	g.Expect(hashes).To(ConsistOf("aaa", "bbb"))
}

func TestWalkCommits_Empty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &Directory{
		Name:  "root",
		Files: []*File{{Name: "a.go"}},
	}

	var count int

	WalkCommits(root, func(_ *Commit, _ *File) { count++ })

	g.Expect(count).To(Equal(0))
}
