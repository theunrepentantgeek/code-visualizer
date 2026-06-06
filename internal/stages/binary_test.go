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

	c := &stages.CommonState{Root: root}

	g.Expect(stages.FilterBinaryFiles(c, true)).To(Succeed())
	g.Expect(root.Files).To(HaveLen(2))
}

func TestFilterBinaryFiles_AllBinary_ReturnsNoFilesError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Files: []*model.File{
			{Name: "a.bin", IsBinary: true},
			{Name: "b.bin", IsBinary: true},
		},
	}

	c := &stages.CommonState{Root: root}
	err := stages.FilterBinaryFiles(c, false)

	var nfe *stages.NoFilesAfterFilterError
	g.Expect(errors.As(err, &nfe)).To(BeTrue())
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
