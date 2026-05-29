package walk_test

import (
"testing"

. "github.com/onsi/gomega"

"github.com/theunrepentantgeek/code-visualizer/internal/model"
"github.com/theunrepentantgeek/code-visualizer/internal/walk"
)

func TestFiles_FlatDir(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

root := &model.Directory{
Path:  "/root",
Name:  "root",
Files: []*model.File{{Path: "/root/a.go", Name: "a.go"}, {Path: "/root/b.go", Name: "b.go"}},
}

var names []string

walk.Files(root, func(f *model.File) { names = append(names, f.Name) })

g.Expect(names).To(ConsistOf("a.go", "b.go"))
}

func TestFiles_Nested(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

child := &model.Directory{
Path:  "/root/sub",
Name:  "sub",
Files: []*model.File{{Path: "/root/sub/c.go", Name: "c.go"}},
}
root := &model.Directory{
Path:  "/root",
Name:  "root",
Files: []*model.File{{Path: "/root/a.go", Name: "a.go"}},
Dirs:  []*model.Directory{child},
}

var names []string

walk.Files(root, func(f *model.File) { names = append(names, f.Name) })

g.Expect(names).To(ConsistOf("a.go", "c.go"))
}

func TestFiles_Empty(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

root := &model.Directory{Path: "/root", Name: "root"}

var count int

walk.Files(root, func(_ *model.File) { count++ })

g.Expect(count).To(Equal(0))
}

func TestFiles_DeepNesting(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

deep := &model.Directory{
Path:  "/root/a/b",
Name:  "b",
Files: []*model.File{{Path: "/root/a/b/leaf.go", Name: "leaf.go"}},
}
mid := &model.Directory{
Path:  "/root/a",
Name:  "a",
Files: []*model.File{{Path: "/root/a/mid.go", Name: "mid.go"}},
Dirs:  []*model.Directory{deep},
}
root := &model.Directory{
Path:  "/root",
Name:  "root",
Files: []*model.File{{Path: "/root/root.go", Name: "root.go"}},
Dirs:  []*model.Directory{mid},
}

var names []string

walk.Files(root, func(f *model.File) { names = append(names, f.Name) })

g.Expect(names).To(ConsistOf("root.go", "mid.go", "leaf.go"))
}

func TestDirectories_FlatDir(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

root := &model.Directory{Path: "/root", Name: "root"}

var names []string

walk.Directories(root, func(d *model.Directory) { names = append(names, d.Name) })

g.Expect(names).To(Equal([]string{"root"}))
}

func TestDirectories_IncludesRoot(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

child := &model.Directory{Path: "/root/sub", Name: "sub"}
root := &model.Directory{Path: "/root", Name: "root", Dirs: []*model.Directory{child}}

var names []string

walk.Directories(root, func(d *model.Directory) { names = append(names, d.Name) })

g.Expect(names).To(ConsistOf("root", "sub"))
}

func TestDirectories_PostOrder(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

a := &model.Directory{Path: "/root/a", Name: "a"}
b := &model.Directory{Path: "/root/b", Name: "b"}
root := &model.Directory{Path: "/root", Name: "root", Dirs: []*model.Directory{a, b}}

var names []string

walk.Directories(root, func(d *model.Directory) { names = append(names, d.Name) })

g.Expect(names).To(HaveLen(3))

if names != nil {
g.Expect(names[len(names)-1]).To(Equal("root"), "root must be visited last (post-order)")
}
}

func TestDirectories_DeepNesting_PostOrder(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

leaf := &model.Directory{Path: "/root/a/b", Name: "b"}
mid := &model.Directory{Path: "/root/a", Name: "a", Dirs: []*model.Directory{leaf}}
root := &model.Directory{Path: "/root", Name: "root", Dirs: []*model.Directory{mid}}

var names []string

walk.Directories(root, func(d *model.Directory) { names = append(names, d.Name) })

g.Expect(names).To(Equal([]string{"b", "a", "root"}))
}

func TestDirectories_Empty(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

root := &model.Directory{Path: "/root", Name: "root"}

var count int

walk.Directories(root, func(_ *model.Directory) { count++ })

g.Expect(count).To(Equal(1))
}

func TestCountFiles_Empty(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

root := &model.Directory{Path: "/root", Name: "root"}

g.Expect(walk.CountFiles(root)).To(Equal(0))
}

func TestCountFiles_FlatDir(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

root := &model.Directory{
Path:  "/root",
Name:  "root",
Files: []*model.File{{Name: "a.go"}, {Name: "b.go"}},
}

g.Expect(walk.CountFiles(root)).To(Equal(2))
}

func TestCountFiles_Nested(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

child := &model.Directory{
Path:  "/root/sub",
Name:  "sub",
Files: []*model.File{{Name: "c.go"}},
}
root := &model.Directory{
Path:  "/root",
Name:  "root",
Files: []*model.File{{Name: "a.go"}},
Dirs:  []*model.Directory{child},
}

g.Expect(walk.CountFiles(root)).To(Equal(2))
}

func TestCountDirs_Empty(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

root := &model.Directory{Path: "/root", Name: "root"}

g.Expect(walk.CountDirs(root)).To(Equal(0))
}

func TestCountDirs_FlatDirs(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

a := &model.Directory{Path: "/root/a", Name: "a"}
b := &model.Directory{Path: "/root/b", Name: "b"}
root := &model.Directory{Path: "/root", Name: "root", Dirs: []*model.Directory{a, b}}

g.Expect(walk.CountDirs(root)).To(Equal(2))
}

func TestCountDirs_Nested(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

leaf := &model.Directory{Path: "/root/a/b", Name: "b"}
mid := &model.Directory{Path: "/root/a", Name: "a", Dirs: []*model.Directory{leaf}}
root := &model.Directory{Path: "/root", Name: "root", Dirs: []*model.Directory{mid}}

g.Expect(walk.CountDirs(root)).To(Equal(2))
}
