package bubbletree

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestBubbleNodeIndexEmptyNode(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	root := &BubbleNode{Path: "root", IsDirectory: true}

	dirs, files := root.Index()

	g.Expect(dirs).To(gomega.HaveLen(0))
	g.Expect(files).To(gomega.HaveLen(0))
	g.Expect(dirs).NotTo(gomega.HaveKey("root"))
	g.Expect(files).NotTo(gomega.HaveKey("root"))
}

func TestBubbleNodeIndexOnlyFileChildren(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	root := &BubbleNode{
		Path:        "root",
		IsDirectory: true,
		Children: []BubbleNode{
			{Path: "root/a.go", Label: "a.go"},
			{Path: "root/b.go", Label: "b.go"},
		},
	}

	dirs, files := root.Index()

	g.Expect(dirs).To(gomega.HaveLen(0))
	g.Expect(files).To(gomega.HaveLen(2))
	g.Expect(files).To(gomega.HaveKey("root/a.go"))
	g.Expect(files).To(gomega.HaveKey("root/b.go"))
	g.Expect(files["root/a.go"]).To(gomega.BeIdenticalTo(&root.Children[0]))
	g.Expect(files["root/b.go"]).To(gomega.BeIdenticalTo(&root.Children[1]))
	g.Expect(files).NotTo(gomega.HaveKey("root"))
}

func TestBubbleNodeIndexOnlyDirectoryChildren(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	root := &BubbleNode{
		Path:        "root",
		IsDirectory: true,
		Children: []BubbleNode{
			{Path: "root/dir-a", Label: "dir-a", IsDirectory: true},
			{Path: "root/dir-b", Label: "dir-b", IsDirectory: true},
		},
	}

	dirs, files := root.Index()

	g.Expect(dirs).To(gomega.HaveLen(2))
	g.Expect(files).To(gomega.HaveLen(0))
	g.Expect(dirs).To(gomega.HaveKey("root/dir-a"))
	g.Expect(dirs).To(gomega.HaveKey("root/dir-b"))
	g.Expect(dirs["root/dir-a"]).To(gomega.BeIdenticalTo(&root.Children[0]))
	g.Expect(dirs["root/dir-b"]).To(gomega.BeIdenticalTo(&root.Children[1]))
	g.Expect(dirs).NotTo(gomega.HaveKey("root"))
}

func TestBubbleNodeIndexMixedDirectChildren(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	root := &BubbleNode{
		Path:        "root",
		IsDirectory: true,
		Children: []BubbleNode{
			{Path: "root/a.go", Label: "a.go"},
			{Path: "root/docs", Label: "docs", IsDirectory: true},
			{Path: "root/b.go", Label: "b.go"},
		},
	}

	dirs, files := root.Index()

	g.Expect(dirs).To(gomega.HaveLen(1))
	g.Expect(files).To(gomega.HaveLen(2))
	g.Expect(dirs["root/docs"]).To(gomega.BeIdenticalTo(&root.Children[1]))
	g.Expect(files["root/a.go"]).To(gomega.BeIdenticalTo(&root.Children[0]))
	g.Expect(files["root/b.go"]).To(gomega.BeIdenticalTo(&root.Children[2]))
}

func TestBubbleNodeIndexNestedDirectoriesIncludesDescendants(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	root := &BubbleNode{
		Path:        "root",
		IsDirectory: true,
		Children: []BubbleNode{
			{
				Path:        "root/src",
				Label:       "src",
				IsDirectory: true,
				Children: []BubbleNode{
					{Path: "root/src/main.go", Label: "main.go"},
					{
						Path:        "root/src/internal",
						Label:       "internal",
						IsDirectory: true,
						Children: []BubbleNode{
							{Path: "root/src/internal/helper.go", Label: "helper.go"},
						},
					},
				},
			},
			{Path: "root/README.md", Label: "README.md"},
		},
	}

	dirs, files := root.Index()

	g.Expect(dirs).To(gomega.HaveLen(2))
	g.Expect(files).To(gomega.HaveLen(3))
	g.Expect(dirs["root/src"]).To(gomega.BeIdenticalTo(&root.Children[0]))
	g.Expect(dirs["root/src/internal"]).To(gomega.BeIdenticalTo(&root.Children[0].Children[1]))
	g.Expect(files["root/src/main.go"]).To(gomega.BeIdenticalTo(&root.Children[0].Children[0]))
	g.Expect(files["root/src/internal/helper.go"]).To(gomega.BeIdenticalTo(&root.Children[0].Children[1].Children[0]))
	g.Expect(files["root/README.md"]).To(gomega.BeIdenticalTo(&root.Children[1]))
	g.Expect(dirs).NotTo(gomega.HaveKey("root"))
	g.Expect(files).NotTo(gomega.HaveKey("root"))
}

func TestBubbleNodeIndexReturnedPointersReferenceOriginalNodes(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	root := &BubbleNode{
		Path:        "root",
		IsDirectory: true,
		Children: []BubbleNode{
			{Path: "root/docs", Label: "docs", IsDirectory: true},
			{Path: "root/main.go", Label: "main.go"},
		},
	}

	dirs, files := root.Index()

	dirs["root/docs"].ShowLabel = true
	files["root/main.go"].Label = "renamed.go"

	g.Expect(root.Children[0].ShowLabel).To(gomega.BeTrue())
	g.Expect(root.Children[1].Label).To(gomega.Equal("renamed.go"))
}
