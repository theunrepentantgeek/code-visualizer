package stages_test

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// fakeBinaryState satisfies BinaryFilterToggler for these tests.
type fakeBinaryState struct {
	common     stages.CommonState
	includeBin bool
}

func (f *fakeBinaryState) Common() *stages.CommonState { return &f.common }
func (f *fakeBinaryState) IncludeBinary() bool         { return f.includeBin }

func TestFilterBinaryFiles_IncludeFlagSet_NoOp(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Files: []*model.File{{Name: "a.bin"}, {Name: "b.go"}},
	}

	s := &fakeBinaryState{
		common:     stages.CommonState{Root: root},
		includeBin: true,
	}

	g.Expect(stages.FilterBinaryFiles[*fakeBinaryState](s)).To(Succeed())
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

	s := &fakeBinaryState{common: stages.CommonState{Root: root}}
	err := stages.FilterBinaryFiles[*fakeBinaryState](s)

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
