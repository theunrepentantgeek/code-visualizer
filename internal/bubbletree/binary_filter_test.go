package bubbletree_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/bubbletree"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestFilterBinaryFiles_RespectsIncludeFlag(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Files: []*model.File{{Name: "a.bin", IsBinary: true}, {Name: "b.go"}},
	}
	common := &stages.CommonState{Root: root}
	viz := &bubbletree.State{IncludeBinaryFiles: true}

	g.Expect(bubbletree.FilterBinaryFiles(common, viz)).To(Succeed())
	g.Expect(root.Files).To(HaveLen(2))
}

func TestFilterBinaryFiles_DefaultStripsBinary(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Files: []*model.File{{Name: "a.bin", IsBinary: true}, {Name: "b.go"}},
	}
	common := &stages.CommonState{Root: root}
	viz := &bubbletree.State{IncludeBinaryFiles: false}

	g.Expect(bubbletree.FilterBinaryFiles(common, viz)).To(Succeed())
	g.Expect(root.Files).To(HaveLen(1))
	g.Expect(root.Files[0].Name).To(Equal("b.go"))
}
