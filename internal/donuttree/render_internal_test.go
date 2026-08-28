package donuttree

import (
	"bytes"
	"encoding/xml"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func renderDonutPipeline(t *testing.T, output string, width, height int) *stages.CommonState {
	t.Helper()
	g := NewGomegaWithT(t)
	cfg := config.New()
	cfg.ImageSize = &config.ImageSize{Width: &width, Height: &height}
	cfg.Title = &config.Title{Text: new("Nested donut")}
	cfg.Footer.Text = new("donut tree test")
	size := "file-lines"
	cfg.DonutTree = &config.DonutTree{
		Size: &size,
		Fill: &config.MetricSpec{Metric: "file-type"},
	}

	root := donutRoot()
	root.Files[0].SetQuantity(filesystem.FileLines, 20)
	root.Files[0].SetClassification(filesystem.FileType, "Markdown")
	root.Dirs[0].Files[0].SetQuantity(filesystem.FileLines, 100)
	root.Dirs[0].Files[0].SetClassification(filesystem.FileType, "Go")

	common := &stages.CommonState{
		Output:     output,
		Flags:      &stages.Flags{Config: cfg},
		RootConfig: cfg,
		Root:       root,
		VizName:    "donut-tree",
	}
	state := pipeline.NewState(common, cfg.DonutTree, &State{})
	pipeline.ApplyFuncX(state, stages.BuildFilterRules)
	pipeline.ApplyFuncX(state, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(state, ResolveMetrics)
	RenderPipeline(state)
	g.Expect(state.Err()).To(Succeed())

	return common
}

func TestRenderDonutPipeline_WritesDecodablePNGAndSVG(t *testing.T) {
	t.Parallel()

	for _, ext := range []string{"png", "svg"} {
		t.Run(ext, func(t *testing.T) {
			t.Parallel()

			g := NewGomegaWithT(t)
			output := filepath.Join(t.TempDir(), "donut."+ext)

			renderDonutPipeline(t, output, 360, 240)

			data, err := os.ReadFile(output)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(data).NotTo(BeEmpty())

			if ext == "png" {
				decoded, format, decodeErr := image.Decode(bytes.NewReader(data))
				g.Expect(decodeErr).NotTo(HaveOccurred())
				g.Expect(format).To(Equal("png"))

				if decoded == nil {
					t.Fatal("decoded PNG is nil")
				}

				g.Expect(decoded.Bounds().Dx()).To(Equal(360))
				g.Expect(decoded.Bounds().Dy()).To(Equal(240))

				return
			}

			var document struct {
				XMLName xml.Name
			}
			g.Expect(xml.Unmarshal(data, &document)).To(Succeed())
			g.Expect(document.XMLName.Local).To(Equal("svg"))
		})
	}
}

func TestRenderDonutPipeline_PreservesNonSquareDimensionsAfterReservations(t *testing.T) {
	t.Parallel()

	const width, height = 800, 480

	g := NewGomegaWithT(t)
	output := filepath.Join(t.TempDir(), "donut.png")

	common := renderDonutPipeline(t, output, width, height)
	g.Expect(common.DrawingBounds.MinY).To(Equal(int(canvas.TitleReservedHeight)))
	g.Expect(common.DrawingBounds.MaxY).To(Equal(height - int(canvas.FooterReservedHeight)))

	file, err := os.Open(output)
	g.Expect(err).NotTo(HaveOccurred())

	defer file.Close()

	decoded, format, err := image.Decode(file)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))

	if decoded == nil {
		t.Fatal("decoded PNG is nil")
	}

	g.Expect(decoded.Bounds().Dx()).To(Equal(width))
	g.Expect(decoded.Bounds().Dy()).To(Equal(height))
}
