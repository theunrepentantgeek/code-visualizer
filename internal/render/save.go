package render

import (
	"image/jpeg"
	"os"

	"github.com/fogleman/gg"
	"github.com/rotisserie/eris"
)

const jpegQuality = 95

// saveContextPNG saves the gg context as a PNG file.
func saveContextPNG(dc *gg.Context, path string) error {
	return eris.Wrap(dc.SavePNG(path), "failed to save PNG")
}

// saveContextJPG saves the gg context as a JPEG file.
func saveContextJPG(dc *gg.Context, path string) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return eris.Wrap(err, "failed to create JPEG file")
	}

	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = eris.Wrap(closeErr, "failed to close JPEG file")
		}
	}()

	if err := jpeg.Encode(f, dc.Image(), &jpeg.Options{Quality: jpegQuality}); err != nil {
		return eris.Wrap(err, "failed to encode JPEG")
	}

	return nil
}
