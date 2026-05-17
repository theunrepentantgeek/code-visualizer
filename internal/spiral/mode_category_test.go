package spiral

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

const testMetric metric.Name = "test-category"

func makeClassifiedFile(category string) *model.File {
	f := &model.File{Name: category + ".txt"}
	f.SetClassification(testMetric, category)

	return f
}

func TestModeCategory_SingleCategory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []*model.File{
		makeClassifiedFile("go"),
		makeClassifiedFile("go"),
		makeClassifiedFile("go"),
	}

	result := modeCategory(files, testMetric)
	g.Expect(result).To(Equal("go"))
}

func TestModeCategory_ClearWinner(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []*model.File{
		makeClassifiedFile("go"),
		makeClassifiedFile("go"),
		makeClassifiedFile("go"),
		makeClassifiedFile("python"),
		makeClassifiedFile("rust"),
	}

	result := modeCategory(files, testMetric)
	g.Expect(result).To(Equal("go"))
}

func TestModeCategory_TieBreaksLexicographically(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []*model.File{
		makeClassifiedFile("python"),
		makeClassifiedFile("go"),
		makeClassifiedFile("rust"),
	}

	// All three categories have count=1, tie should resolve to "go" (lexicographically first).
	// Run multiple times to confirm determinism.
	for range 100 {
		result := modeCategory(files, testMetric)
		g.Expect(result).To(Equal("go"), "tie-break must be deterministic and lexicographic")
	}
}

func TestModeCategory_EmptyInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := modeCategory(nil, testMetric)
	g.Expect(result).To(BeEmpty())
}

func TestModeCategory_NoClassifications(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := []*model.File{
		{Name: "unclassified.txt"},
	}

	result := modeCategory(files, testMetric)
	g.Expect(result).To(BeEmpty())
}
