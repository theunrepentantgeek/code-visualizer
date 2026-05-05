package render

import (
	"bytes"
	"encoding/xml"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/spiral"
)

// sampleSpiralNodes returns a small deterministic []SpiralNode for render smoke tests.
func sampleSpiralNodes() []spiral.SpiralNode {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	return []spiral.SpiralNode{
		{
			X: 960, Y: 200, DiscRadius: 15,
			Angle: 0, SpiralRadius: 200,
			TimeStart: t0,
			TimeEnd:   t0.Add(time.Hour),
			Label:     "12am", ShowLabel: true,
			FillColour: color.RGBA{R: 100, G: 150, B: 200, A: 255},
		},
		{
			X: 1100, Y: 500, DiscRadius: 20,
			Angle: 1.5, SpiralRadius: 300,
			TimeStart: t0.Add(time.Hour),
			TimeEnd:   t0.Add(2 * time.Hour),
			Label:     "1am", ShowLabel: false,
			FillColour: color.RGBA{R: 200, G: 100, B: 100, A: 255},
		},
		{
			X: 600, Y: 700, DiscRadius: 25,
			Angle: 3.0, SpiralRadius: 400,
			TimeStart: t0.Add(2 * time.Hour),
			TimeEnd:   t0.Add(3 * time.Hour),
			Label:     "2am", ShowLabel: false,
			FillColour:   color.RGBA{R: 150, G: 200, B: 100, A: 255},
			BorderColour: &color.RGBA{R: 50, G: 50, B: 50, A: 255},
		},
	}
}

func TestRenderSpiral_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	nodes := sampleSpiralNodes()
	out := filepath.Join(t.TempDir(), "spiral.png")

	err := RenderSpiral(nodes, 1920, 1920, out, nil)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info).NotTo(BeNil())

	if info != nil {
		g.Expect(info.Size()).To(BeNumerically(">", 0))
	}
}

func TestRenderSpiral_PNG_DecodesAsPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	nodes := sampleSpiralNodes()
	out := filepath.Join(t.TempDir(), "spiral.png")

	err := RenderSpiral(nodes, 1920, 1920, out, nil)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	if f != nil {
		defer f.Close()

		_, imgFmt, decErr := image.Decode(f)
		g.Expect(decErr).NotTo(HaveOccurred())
		g.Expect(imgFmt).To(Equal("png"))
	}
}

func TestRenderSpiral_JPG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	nodes := sampleSpiralNodes()
	out := filepath.Join(t.TempDir(), "spiral.jpg")

	err := RenderSpiral(nodes, 1920, 1920, out, nil)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info).NotTo(BeNil())

	if info != nil {
		g.Expect(info.Size()).To(BeNumerically(">", 0))
	}
}

func TestRenderSpiral_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	nodes := sampleSpiralNodes()
	out := filepath.Join(t.TempDir(), "spiral.svg")

	err := RenderSpiral(nodes, 1920, 1920, out, nil)
	g.Expect(err).NotTo(HaveOccurred())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify it's valid XML with an <svg> root element.
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

func TestRenderSpiral_EmptyNodes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	out := filepath.Join(t.TempDir(), "empty-spiral.png")
	err := RenderSpiral([]spiral.SpiralNode{}, 1920, 1920, out, nil)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info).NotTo(BeNil())

	if info != nil {
		g.Expect(info.Size()).To(BeNumerically(">", 0))
	}
}

func TestRenderSpiral_UnsupportedFormat(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	nodes := sampleSpiralNodes()
	err := RenderSpiral(nodes, 1920, 1920, "spiral.bmp", nil)
	g.Expect(err).To(HaveOccurred())
}
