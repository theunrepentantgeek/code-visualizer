package bubbletree

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestBubbleNodeIndexEmptyNode(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &BubbleNode{Path: "root", IsDirectory: true}

	dirs, files := root.Index()

	g.Expect(dirs).To(BeEmpty())
	g.Expect(files).To(BeEmpty())
	g.Expect(dirs).NotTo(HaveKey("root"))
	g.Expect(files).NotTo(HaveKey("root"))
}

func TestBubbleNodeIndexOnlyFileChildren(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &BubbleNode{
		Path:        "root",
		IsDirectory: true,
		Children: []BubbleNode{
			{Path: "root/a.go", Label: "a.go"},
			{Path: "root/b.go", Label: "b.go"},
		},
	}

	dirs, files := root.Index()

	g.Expect(dirs).To(BeEmpty())
	g.Expect(files).To(HaveLen(2))
	g.Expect(files).To(HaveKey("root/a.go"))
	g.Expect(files).To(HaveKey("root/b.go"))
	g.Expect(files["root/a.go"]).To(BeIdenticalTo(&root.Children[0]))
	g.Expect(files["root/b.go"]).To(BeIdenticalTo(&root.Children[1]))
	g.Expect(files).NotTo(HaveKey("root"))
}

func TestBubbleNodeIndexOnlyDirectoryChildren(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &BubbleNode{
		Path:        "root",
		IsDirectory: true,
		Children: []BubbleNode{
			{Path: "root/dir-a", Label: "dir-a", IsDirectory: true},
			{Path: "root/dir-b", Label: "dir-b", IsDirectory: true},
		},
	}

	dirs, files := root.Index()

	g.Expect(dirs).To(HaveLen(2))
	g.Expect(files).To(BeEmpty())
	g.Expect(dirs).To(HaveKey("root/dir-a"))
	g.Expect(dirs).To(HaveKey("root/dir-b"))
	g.Expect(dirs["root/dir-a"]).To(BeIdenticalTo(&root.Children[0]))
	g.Expect(dirs["root/dir-b"]).To(BeIdenticalTo(&root.Children[1]))
	g.Expect(dirs).NotTo(HaveKey("root"))
}

func TestBubbleNodeIndexMixedDirectChildren(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

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

	g.Expect(dirs).To(HaveLen(1))
	g.Expect(files).To(HaveLen(2))
	g.Expect(dirs["root/docs"]).To(BeIdenticalTo(&root.Children[1]))
	g.Expect(files["root/a.go"]).To(BeIdenticalTo(&root.Children[0]))
	g.Expect(files["root/b.go"]).To(BeIdenticalTo(&root.Children[2]))
}

func TestBubbleNodeIndexNestedDirectoriesIncludesDescendants(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

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

	g.Expect(dirs).To(HaveLen(2))
	g.Expect(files).To(HaveLen(3))
	g.Expect(dirs["root/src"]).To(BeIdenticalTo(&root.Children[0]))
	g.Expect(dirs["root/src/internal"]).To(BeIdenticalTo(&root.Children[0].Children[1]))
	g.Expect(files["root/src/main.go"]).To(BeIdenticalTo(&root.Children[0].Children[0]))
	g.Expect(files["root/src/internal/helper.go"]).To(BeIdenticalTo(&root.Children[0].Children[1].Children[0]))
	g.Expect(files["root/README.md"]).To(BeIdenticalTo(&root.Children[1]))
	g.Expect(dirs).NotTo(HaveKey("root"))
	g.Expect(files).NotTo(HaveKey("root"))
}

func TestBubbleNodeIndexReturnedPointersReferenceOriginalNodes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &BubbleNode{
		Path:        "root",
		IsDirectory: true,
		Children: []BubbleNode{
			{Path: "root/docs", Label: "docs", IsDirectory: true},
			{Path: "root/main.go", Label: "main.go"},
		},
	}

	dirs, files := root.Index()

	dir, dirOK := dirs["root/docs"]
	g.Expect(dirOK).To(BeTrue())

	if dir == nil {
		t.Fatal("expected dirs[\"root/docs\"] to be non-nil")
	}

	dir.ShowLabel = true

	file, fileOK := files["root/main.go"]
	g.Expect(fileOK).To(BeTrue())

	if file == nil {
		t.Fatal("expected files[\"root/main.go\"] to be non-nil")
	}

	file.Label = "renamed.go"

	g.Expect(root.Children[0].ShowLabel).To(BeTrue())
	g.Expect(root.Children[1].Label).To(Equal("renamed.go"))
}
