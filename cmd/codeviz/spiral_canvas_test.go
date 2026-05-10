package main

import (
	"bytes"
	"encoding/xml"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/spiral"
)

func makeSpiralTestFile(name, ext string, size int64) *model.File {
	f := &model.File{Name: name, Extension: ext}
	f.SetQuantity(filesystem.FileSize, size)
	f.SetClassification(filesystem.FileType, ext)

	return f
}

func sampleTimeBuckets() []spiral.TimeBucket {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	return []spiral.TimeBucket{
		{
			Start: t0, End: t0.Add(time.Hour),
			Files: []*model.File{
				makeSpiralTestFile("a.go", "go", 100),
				makeSpiralTestFile("b.go", "go", 200),
			},
			SizeValue: 300, FillValue: 300, FillLabel: "go",
		},
		{
			Start: t0.Add(time.Hour), End: t0.Add(2 * time.Hour),
			Files: []*model.File{
				makeSpiralTestFile("c.py", "py", 50),
			},
			SizeValue: 50, FillValue: 50, FillLabel: "py",
		},
		{
			Start: t0.Add(2 * time.Hour), End: t0.Add(3 * time.Hour),
			Files:     []*model.File{},
			SizeValue: 0,
		},
	}
}

func TestSpiralBorderWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(spiralBorderWidth(7.9)).To(Equal(2.0))
	g.Expect(spiralBorderWidth(8.0)).To(Equal(3.0))
	g.Expect(spiralBorderWidth(10.0)).To(Equal(3.0))
}

func TestBuildSpiralInks_Numeric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	inks := buildSpiralInks(
		buckets,
		filesystem.FileSize,
		palette.Temperature,
		"",
		"",
	)

	g.Expect(inks.fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.border.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestBuildSpiralInks_Categorical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	inks := buildSpiralInks(
		buckets,
		filesystem.FileType,
		palette.Categorization,
		"",
		"",
	)

	g.Expect(inks.fill.Info().Kind).To(Equal(canvas.InkCategorical))
}

func TestBuildSpiralInks_NoMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	inks := buildSpiralInks(buckets, "", "", "", "")

	g.Expect(inks.fill.Info().Kind).To(Equal(canvas.InkFixed))
	g.Expect(inks.border.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestRenderSpiralToCanvas_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	nodes := spiral.Layout(buckets, 800, 600, spiral.Hourly, spiral.LabelNone)
	inks := buildSpiralInks(buckets, "", "", "", "")
	cv := renderSpiralToCanvas(nodes, buckets, 800, 600, inks)

	out := filepath.Join(t.TempDir(), "spiral.png")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
}

func TestRenderSpiralToCanvas_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	nodes := spiral.Layout(buckets, 400, 300, spiral.Hourly, spiral.LabelNone)
	inks := buildSpiralInks(buckets, "", "", "", "")
	cv := renderSpiralToCanvas(nodes, buckets, 400, 300, inks)

	out := filepath.Join(t.TempDir(), "spiral.svg")
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

func TestRenderSpiralToCanvas_JPG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	nodes := spiral.Layout(buckets, 400, 300, spiral.Hourly, spiral.LabelNone)
	inks := buildSpiralInks(buckets, "", "", "", "")
	cv := renderSpiralToCanvas(nodes, buckets, 400, 300, inks)

	out := filepath.Join(t.TempDir(), "spiral.jpg")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("jpeg"))
}
