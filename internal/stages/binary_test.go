package stages_test

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestFilterBinaryFiles_IncludeFlagSet_NoOp(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Files: []*model.File{{Name: "a.bin"}, {Name: "b.go"}},
	}

	c := &stages.CommonState{Root: root, IncludeBinaryFiles: true}

	g.Expect(stages.FilterBinaryFiles(c)).To(Succeed())
	g.Expect(root.Files).To(HaveLen(2))
}

func TestFilterBinaryFiles_NoFilesInRoot_ReturnsNoFilesError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Binary files are excluded during scanning; an empty root means all files
	// were binary.
	root := &model.Directory{}

	c := &stages.CommonState{Root: root}
	err := stages.FilterBinaryFiles(c)

	var nfe *stages.NoFilesAfterFilterError
	g.Expect(errors.As(err, &nfe)).To(BeTrue())
}

func TestFilterBinaryFiles_TextFilesPresent_Succeeds(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Binary files have already been excluded during scanning; the root only
	// contains text files.
	root := &model.Directory{
		Files: []*model.File{{Name: "b.go"}},
	}

	c := &stages.CommonState{Root: root}

	g.Expect(stages.FilterBinaryFiles(c)).To(Succeed())
	g.Expect(root.Files).To(HaveLen(1))
}

func TestCountAll_NestedDirs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Files: []*model.File{{Name: "a"}, {Name: "b"}},
		Dirs: []*model.Directory{
			{Files: []*model.File{{Name: "c"}}},
		},
	}

	files, dirs := stages.CountAll(root)
	g.Expect(files).To(Equal(3))
	g.Expect(dirs).To(Equal(1))
}
