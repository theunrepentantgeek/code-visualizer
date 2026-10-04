package model

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestDirectorySelection_WalkFiles_SubtreeIncludesDescendants(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	root := selectionTestTree()

	var names []string

	err := (DirectorySelection{
		Directory: root,
		Scope:     DirectorySubtree,
	}).WalkFiles(func(file *File) {
		names = append(names, file.Name)
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(names).To(ConsistOf("root.go", "internal.go", "api.go", "private.go"))
}

func TestDirectorySelection_WalkFiles_DirectFilesExcludesDescendants(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	root := selectionTestTree()

	var names []string

	err := (DirectorySelection{
		Directory: root,
		Scope:     DirectoryDirectFiles,
	}).WalkFiles(func(file *File) {
		names = append(names, file.Name)
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(names).To(Equal([]string{"root.go"}))
}

func TestDirectorySelection_WalkFiles_RejectsInvalidScope(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := (DirectorySelection{Directory: selectionTestTree()}).WalkFiles(func(*File) {})

	g.Expect(err).To(MatchError(ContainSubstring("invalid directory scope")))
}

func TestPartitionDirectories_PartitionsRootDirectFilesAndChildren(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	root := selectionTestTree()

	partition := PartitionDirectories(root, nil)

	g.Expect(partition).To(HaveLen(2))
	g.Expect(partition["."].Directory).To(BeIdenticalTo(root))
	g.Expect(partition["."].Scope).To(Equal(DirectoryDirectFiles))
	g.Expect(partition["internal"].Directory).To(BeIdenticalTo(root.Dirs[0]))
	g.Expect(partition["internal"].Scope).To(Equal(DirectorySubtree))
}

func TestPartitionDirectories_RecursivelyPartitionsExpandedDirectories(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	root := selectionTestTree()

	partition := PartitionDirectories(root, []string{"internal", "internal/api"})

	g.Expect(partition).To(HaveLen(4))
	g.Expect(partition["."].Scope).To(Equal(DirectoryDirectFiles))
	g.Expect(partition["internal"].Scope).To(Equal(DirectoryDirectFiles))
	g.Expect(partition["internal/api"].Scope).To(Equal(DirectoryDirectFiles))
	g.Expect(partition["internal/api/private"].Scope).To(Equal(DirectorySubtree))
}

func TestPartitionDirectories_OmitsEmptyDirectFileSelections(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	root := selectionTestTree()
	root.Dirs[0].Files = nil

	partition := PartitionDirectories(root, []string{"internal"})

	g.Expect(partition).NotTo(HaveKey("internal"))
	g.Expect(partition).To(HaveKey("internal/api"))
}

func TestPartitionDirectories_IgnoresUnreachableNestedExpansion(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	partition := PartitionDirectories(selectionTestTree(), []string{"internal/api"})

	g.Expect(partition).To(HaveKey("internal"))
	g.Expect(partition).NotTo(HaveKey("internal/api"))
}

func TestPartitionDirectories_NilAndFileEmptyRootsReturnEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(PartitionDirectories(nil, nil)).To(BeEmpty())
	g.Expect(PartitionDirectories(&Directory{RepoPath: "."}, nil)).To(BeEmpty())
}

func TestPartitionDirectories_AssignsEveryFileExactlyOnce(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	root := selectionTestTree()

	partition := PartitionDirectories(root, []string{"internal", "internal/api"})
	names := make([]string, 0)

	for _, selection := range partition {
		g.Expect(selection.WalkFiles(func(file *File) {
			names = append(names, file.Name)
		})).To(Succeed())
	}

	g.Expect(names).To(ConsistOf("root.go", "internal.go", "api.go", "private.go"))
}

func selectionTestTree() *Directory {
	private := &Directory{
		RepoPath: "internal/api/private",
		Files:    []*File{{Name: "private.go"}},
	}
	api := &Directory{
		RepoPath: "internal/api",
		Files:    []*File{{Name: "api.go"}},
		Dirs:     []*Directory{private},
	}
	internal := &Directory{
		RepoPath: "internal",
		Files:    []*File{{Name: "internal.go"}},
		Dirs:     []*Directory{api},
	}

	return &Directory{
		RepoPath: ".",
		Files:    []*File{{Name: "root.go"}},
		Dirs:     []*Directory{internal},
	}
}
