package spiral_test

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

	"github.com/theunrepentantgeek/code-visualizer/internal/spiral"
)

func TestRenderToCanvas_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	layout := spiral.Layout(buckets, 800, 600, spiral.Hourly, spiral.LabelNone)
	inks := spiral.BuildInks(buckets, "", "", "", "")
	cv := spiral.RenderToCanvas(layout, buckets, 800, 600, inks)

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

func TestRenderToCanvas_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	layout := spiral.Layout(buckets, 400, 300, spiral.Hourly, spiral.LabelNone)
	inks := spiral.BuildInks(buckets, "", "", "", "")
	cv := spiral.RenderToCanvas(layout, buckets, 400, 300, inks)

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

func TestRenderToCanvas_JPG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	layout := spiral.Layout(buckets, 400, 300, spiral.Hourly, spiral.LabelNone)
	inks := spiral.BuildInks(buckets, "", "", "", "")
	cv := spiral.RenderToCanvas(layout, buckets, 400, 300, inks)

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
