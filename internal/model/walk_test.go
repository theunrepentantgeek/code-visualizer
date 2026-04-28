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
