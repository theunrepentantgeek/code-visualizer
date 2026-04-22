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
