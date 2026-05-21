package treemap_test

import (
	"bytes"
	"encoding/xml"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
)

func TestMain(m *testing.M) {
	filesystem.Register()
	m.Run()
}

func makeTestFile(name, ext string, size int64) *model.File {
	f := &model.File{Name: name, Extension: ext}
	f.SetQuantity(filesystem.FileSize, size)
	f.SetClassification(filesystem.FileType, ext)

	return f
}

func TestTreemapDynBorderWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(treemap.DynBorderWidth(10, 10, canvas.InkFixed)).To(Equal(0.5))
	g.Expect(treemap.DynBorderWidth(10, 10, canvas.InkNumeric)).To(Equal(1.0))
	g.Expect(treemap.DynBorderWidth(50, 50, canvas.InkNumeric)).To(Equal(2.0))
	g.Expect(treemap.DynBorderWidth(200, 200, canvas.InkNumeric)).To(Equal(3.0))
}

func TestBuildTreemapInks_Numeric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeTestFile("a.go", "go", 100),
			makeTestFile("b.go", "go", 200),
		},
	}

	inks := treemap.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.Border.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestBuildTreemapInks_Categorical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeTestFile("a.go", "go", 100),
			makeTestFile("b.rs", "rs", 200),
		},
	}

	inks := treemap.BuildInks(root, filesystem.FileType, palette.Categorization, "", "")

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkCategorical))
}

func TestRenderTreemapToCanvas_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "flat",
		Files: []*model.File{
			makeTestFile("small.txt", "txt", 5),
			makeTestFile("medium.go", "go", 100),
			makeTestFile("large.rs", "rs", 1000),
		},
	}

	rects := treemap.Layout(root, 800, 600, filesystem.FileSize)
	inks := treemap.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := treemap.RenderToCanvas(rects, root, 800, 600, inks, "")

	out := filepath.Join(t.TempDir(), "treemap.png")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
}

func TestRenderTreemapToCanvas_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "flat",
		Files: []*model.File{
			makeTestFile("a.go", "go", 100),
			makeTestFile("b.go", "go", 200),
		},
	}

	rects := treemap.Layout(root, 400, 300, filesystem.FileSize)
	inks := treemap.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := treemap.RenderToCanvas(rects, root, 400, 300, inks, "")

	out := filepath.Join(t.TempDir(), "treemap.svg")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	decoder := xml.NewDecoder(bytes.NewReader(data))

	var rootElement string

	for {
		tok, xmlErr := decoder.Token()
		if xmlErr != nil {
			break
		}

		if se, ok := tok.(xml.StartElement); ok {
			rootElement = se.Name.Local

			break
		}
	}

	g.Expect(rootElement).To(Equal("svg"))
}

func TestRenderTreemapToCanvas_JPG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "flat",
		Files: []*model.File{
			makeTestFile("a.go", "go", 100),
		},
	}

	rects := treemap.Layout(root, 400, 300, filesystem.FileSize)
	inks := treemap.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := treemap.RenderToCanvas(rects, root, 400, 300, inks, "")

	out := filepath.Join(t.TempDir(), "treemap.jpg")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("jpeg"))
}
