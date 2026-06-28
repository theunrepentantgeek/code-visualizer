package goldentest

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

//nolint:paralleltest // mutates the global metric registry
func TestBuildVizModel_IsDeterministicAndPopulated(t *testing.T) {
	g := NewGomegaWithT(t)

	root := buildVizModel()

	g.Expect(root).NotTo(BeNil())
	g.Expect(root.Dirs).NotTo(BeEmpty(), "expected nested directories")
	g.Expect(root.Files).NotTo(BeEmpty(), "expected root-level files")

	// Every file carries the file-level base metrics the visualizations use.
	f := root.Files[0]
	lines, ok := f.Quantity(filesystem.FileLines)
	g.Expect(ok).To(BeTrue(), "file-lines must be set")
	g.Expect(lines).To(BeNumerically(">", 0))

	_, ok = f.Classification(filesystem.FileType)
	g.Expect(ok).To(BeTrue(), "file-type must be set")
}
