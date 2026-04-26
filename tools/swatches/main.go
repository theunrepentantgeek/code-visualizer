// Command swatches generates PNG colour-bar images for every registered palette.
//
// Usage:
//
//	go run ./tools/swatches [output-dir]
//
// If output-dir is omitted it defaults to docs/.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/bevan/code-visualizer/internal/palette"
)

const (
	stepWidth = 60
	height    = 30
	border    = 1
)

func main() {
	outDir := "docs"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}

	for _, name := range palette.Names() {
		p := palette.GetPalette(name)
		if len(p.Colours) == 0 {
			fmt.Fprintf(os.Stderr, "skipping empty palette %s\n", name)

			continue
		}

		if err := writeSwatch(outDir, p); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
}

func writeSwatch(outDir string, p palette.ColourPalette) error {
	cleanDir := filepath.Clean(outDir)

	if info, err := os.Stat(cleanDir); err != nil || !info.IsDir() {
		return fmt.Errorf("output directory does not exist: %s", cleanDir)
	}

	n := len(p.Colours)
	totalWidth := n*stepWidth + (n+1)*border
	totalHeight := height + 2*border

	img := createSwatchImage(p.Colours, totalWidth, totalHeight)

	path := filepath.Join(cleanDir, fmt.Sprintf("palette-%s.png", p.Name))

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}

	fmt.Printf("wrote %s (%dx%d)\n", path, totalWidth, totalHeight)

	return nil
}

func createSwatchImage(colours []color.RGBA, totalWidth, totalHeight int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, totalWidth, totalHeight))

	borderColour := color.RGBA{R: 80, G: 80, B: 80, A: 255}

	for y := range totalHeight {
		for x := range totalWidth {
			img.Set(x, y, borderColour)
		}
	}

	for i, c := range colours {
		x0 := border + i*(stepWidth+border)
		for y := border; y < border+height; y++ {
			for x := x0; x < x0+stepWidth; x++ {
				img.Set(x, y, c)
			}
		}
	}

	return img
}
