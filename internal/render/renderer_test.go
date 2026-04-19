package render

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/sebdah/goldie/v2"

	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
	"github.com/bevan/code-visualizer/internal/scan"
	"github.com/bevan/code-visualizer/internal/treemap"
)

func makeFile(name, ext string, size int64) *model.File {
	f := &model.File{Name: name, Extension: ext}
	f.SetQuantity(filesystem.FileSize, size)
	f.SetClassification(filesystem.FileType, ext)

	return f
}

func TestRenderFlatDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "flat",
		Files: []*model.File{
			makeFile("small.txt", "txt", 5),
			makeFile("medium.go", "go", 100),
			makeFile("large.rs", "rs", 1000),
		},
	}

	rects := treemap.Layout(root, 800, 600, filesystem.FileSize)
	out := filepath.Join(t.TempDir(), "flat.png")
	err := Render(rects, 800, 600, out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info).NotTo(BeNil())

	if info == nil {
		return
	}

	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestRenderNestedDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "nested",
		Files: []*model.File{
			makeFile("root.txt", "txt", 50),
		},
		Dirs: []*model.Directory{
			{
				Name: "sub",
				Files: []*model.File{
					makeFile("child.go", "go", 200),
				},
			},
		},
	}

	rects := treemap.Layout(root, 800, 600, filesystem.FileSize)
	out := filepath.Join(t.TempDir(), "nested.png")
	err := Render(rects, 800, 600, out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info).NotTo(BeNil())

	if info == nil {
		return
	}

	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestRenderWithBorderColour(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	blue := color.RGBA{R: 0, G: 0, B: 255, A: 255}
	green := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	rects := treemap.TreemapRectangle{
		X: 0, Y: 0, W: 800, H: 600,
		Label: "root", IsDirectory: true,
		Children: []treemap.TreemapRectangle{
			{X: 4, Y: 20, W: 380, H: 576, Label: "a.go", FillColour: red, BorderColour: &blue},
			{X: 388, Y: 20, W: 380, H: 576, Label: "b.go", FillColour: green, BorderColour: &red},
		},
	}

	out := filepath.Join(t.TempDir(), "border.png")
	err := Render(rects, 800, 600, out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info).NotTo(BeNil())

	if info == nil {
		return
	}

	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestRenderNoBorderWhenNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rects := treemap.TreemapRectangle{
		X: 0, Y: 0, W: 400, H: 300,
		Label: "root", IsDirectory: true,
		Children: []treemap.TreemapRectangle{
			{
				X: 4, Y: 20, W: 392, H: 276, Label: "a.go",
				FillColour:   color.RGBA{R: 200, G: 200, B: 200, A: 255},
				BorderColour: nil,
			},
		},
	}

	out := filepath.Join(t.TempDir(), "noborder.png")
	err := Render(rects, 400, 300, out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info).NotTo(BeNil())

	if info == nil {
		return
	}

	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

// paletteTreemap builds a deterministic treemap with files coloured by the given palette.
func paletteTreemap(p palette.ColourPalette) treemap.TreemapRectangle {
	n := len(p.Colours)
	children := make([]treemap.TreemapRectangle, n)

	w := 800.0 / float64(n)
	for i := range n {
		children[i] = treemap.TreemapRectangle{
			X: float64(i) * w, Y: 20, W: w, H: 580,
			Label:      "f" + string(rune('0'+i%10)),
			FillColour: p.Colours[i],
		}
	}

	return treemap.TreemapRectangle{
		X: 0, Y: 0, W: 800, H: 600,
		Label: "root", IsDirectory: true,
		Children: children,
	}
}

func TestGoldenFile_NeutralPalette(t *testing.T) {
	t.Parallel()
	goldenPaletteTest(t, palette.Neutral, "neutral-palette")
}

func TestGoldenFile_CategorizationPalette(t *testing.T) {
	t.Parallel()
	goldenPaletteTest(t, palette.Categorization, "categorization-palette")
}

func TestGoldenFile_TemperaturePalette(t *testing.T) {
	t.Parallel()
	goldenPaletteTest(t, palette.Temperature, "temperature-palette")
}

func TestGoldenFile_GoodBadPalette(t *testing.T) {
	t.Parallel()
	goldenPaletteTest(t, palette.GoodBad, "goodbad-palette")
}

func goldenPaletteTest(t *testing.T, name palette.PaletteName, fixtureName string) {
	t.Helper()
	g := NewGomegaWithT(t)

	p := palette.GetPalette(name)
	root := paletteTreemap(p)
	out := filepath.Join(t.TempDir(), fixtureName+".png")
	err := Render(root, 800, 600, out)
	g.Expect(err).NotTo(HaveOccurred())

	actual, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	gld := goldie.New(t, goldie.WithFixtureDir("testdata"), goldie.WithNameSuffix(".png"))
	gld.Assert(t, fixtureName, actual)
}

// BenchmarkScanAndRender benchmarks the full scan→layout→render pipeline
// with a 1,000-file fixture.
func BenchmarkScanAndRender(b *testing.B) {
	dir := createBenchFixture(b)
	out := filepath.Join(b.TempDir(), "bench.png")

	for b.Loop() {
		root, err := scan.Scan(dir, nil, nil)
		if err != nil {
			b.Fatal(err)
		}

		rects := treemap.Layout(root, 1920, 1080, filesystem.FileSize)
		if err := Render(rects, 1920, 1080, out); err != nil {
			b.Fatal(err)
		}
	}
}

func createBenchFixture(b *testing.B) string {
	b.Helper()

	dir := b.TempDir()

	for d := range 10 {
		subdir := filepath.Join(dir, fmt.Sprintf("dir%02d", d))
		if err := os.MkdirAll(subdir, 0o755); err != nil {
			b.Fatal(err)
		}

		for f := range 100 {
			name := filepath.Join(subdir, fmt.Sprintf("file%03d.go", f))

			data := make([]byte, 100+f*10)
			if err := os.WriteFile(name, data, 0o600); err != nil {
				b.Fatal(err)
			}
		}
	}

	return dir
}

func TestRender_JPG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "flat",
		Files: []*model.File{makeFile("a.go", "go", 100), makeFile("b.go", "go", 200)},
	}

	rects := treemap.Layout(root, 400, 300, filesystem.FileSize)
	out := filepath.Join(t.TempDir(), "output.jpg")

	err := Render(rects, 400, 300, out)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("jpeg"))
}

func TestRender_JPEG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "flat",
		Files: []*model.File{makeFile("a.go", "go", 100)},
	}

	rects := treemap.Layout(root, 400, 300, filesystem.FileSize)
	out := filepath.Join(t.TempDir(), "output.jpeg")

	err := Render(rects, 400, 300, out)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("jpeg"))
}

func TestRender_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "flat",
		Files: []*model.File{
			makeFile("a.go", "go", 100),
			makeFile("b.go", "go", 200),
		},
	}

	rects := treemap.Layout(root, 400, 300, filesystem.FileSize)
	out := filepath.Join(t.TempDir(), "output.svg")

	err := Render(rects, 400, 300, out)
	g.Expect(err).NotTo(HaveOccurred())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify it's valid XML with an <svg> root element
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

	g.Expect(rootElement).To(Equal("svg"), "SVG output should have an <svg> root element")
}

func TestRender_SVG_EscapesLabels(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rects := treemap.TreemapRectangle{
		X: 0, Y: 0, W: 400, H: 300,
		Label: "root", IsDirectory: true,
		Children: []treemap.TreemapRectangle{
			{
				X: 4, Y: 20, W: 392, H: 276, Label: "a&b<c>.go",
				FillColour: color.RGBA{R: 200, G: 200, B: 200, A: 255},
			},
		},
	}

	out := filepath.Join(t.TempDir(), "escape.svg")

	err := Render(rects, 400, 300, out)
	g.Expect(err).NotTo(HaveOccurred())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	// Must be valid XML (will fail if labels aren't escaped)
	var parsed struct{}
	g.Expect(xml.Unmarshal(data, &parsed)).To(Succeed())
}

func TestRender_UnsupportedFormat(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rects := treemap.TreemapRectangle{X: 0, Y: 0, W: 100, H: 100}
	err := Render(rects, 100, 100, "output.bmp")
	g.Expect(err).ToNot(BeNil())

	if err == nil {
		return
	}

	g.Expect(err.Error()).To(ContainSubstring("unsupported image format"))
}

func TestRender_PNG_DecodesAsPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "flat",
		Files: []*model.File{makeFile("a.go", "go", 100)},
	}

	rects := treemap.Layout(root, 400, 300, filesystem.FileSize)
	out := filepath.Join(t.TempDir(), "output.png")

	err := Render(rects, 400, 300, out)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
}
