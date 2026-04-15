package model

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestWalkDirectoriesPostOrder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	leaf1 := &Directory{Path: "/root/a/leaf1", Name: "leaf1"}
	leaf2 := &Directory{Path: "/root/a/leaf2", Name: "leaf2"}
	a := &Directory{Path: "/root/a", Name: "a", Dirs: []*Directory{leaf1, leaf2}}
	b := &Directory{Path: "/root/b", Name: "b"}
	root := &Directory{Path: "/root", Name: "root", Dirs: []*Directory{a, b}}

	var visited []string

	WalkDirectories(root, func(d *Directory) {
		visited = append(visited, d.Name)
	})

	// Post-order: leaves before parents, root last
	g.Expect(visited).To(Equal([]string{"leaf1", "leaf2", "a", "b", "root"}))
}

func TestWalkDirectoriesLeafOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &Directory{Path: "/root", Name: "root"}

	var visited []string

	WalkDirectories(root, func(d *Directory) {
		visited = append(visited, d.Name)
	})

	g.Expect(visited).To(Equal([]string{"root"}))
}

func TestWalkDirectoriesIncludesRoot(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	child := &Directory{Path: "/root/child", Name: "child"}
	root := &Directory{Path: "/root", Name: "root", Dirs: []*Directory{child}}

	var visited []string

	WalkDirectories(root, func(d *Directory) {
		visited = append(visited, d.Name)
	})

	g.Expect(visited).To(ContainElement("root"))
	g.Expect(visited).To(ContainElement("child"))

	if len(visited) < 2 {
		return
	}

	// child must appear before root (post-order)
	g.Expect(visited[0]).To(Equal("child"))
	g.Expect(visited[1]).To(Equal("root"))
}
