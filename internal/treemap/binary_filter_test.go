package treemap_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
)

func TestFilterBinaryFiles_RespectsIncludeFlag(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Files: []*model.File{{Name: "a.bin", IsBinary: true}, {Name: "b.go"}},
	}
	common := &stages.CommonState{Root: root}
	viz := &treemap.State{IncludeBinaryFiles: true}

	g.Expect(treemap.FilterBinaryFiles(common, viz)).To(Succeed())
	g.Expect(root.Files).To(HaveLen(2))
}

func TestFilterBinaryFiles_DefaultStripsBinary(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Files: []*model.File{{Name: "a.bin", IsBinary: true}, {Name: "b.go"}},
	}
	common := &stages.CommonState{Root: root}
	viz := &treemap.State{IncludeBinaryFiles: false}

	g.Expect(treemap.FilterBinaryFiles(common, viz)).To(Succeed())
	g.Expect(root.Files).To(HaveLen(1))
	g.Expect(root.Files[0].Name).To(Equal("b.go"))
}
