package render

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestFormatFromPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		path     string
		expected ImageFormat
	}{
		{"png lowercase", "output.png", FormatPNG},
		{"png uppercase", "output.PNG", FormatPNG},
		{"png mixed case", "output.Png", FormatPNG},
		{"jpg lowercase", "output.jpg", FormatJPG},
		{"jpg uppercase", "output.JPG", FormatJPG},
		{"jpeg lowercase", "output.jpeg", FormatJPG},
		{"jpeg uppercase", "output.JPEG", FormatJPG},
		{"jpeg mixed case", "output.JpEg", FormatJPG},
		{"svg lowercase", "output.svg", FormatSVG},
		{"svg uppercase", "output.SVG", FormatSVG},
		{"svg mixed case", "output.Svg", FormatSVG},
		{"path with dirs", "/tmp/out/chart.png", FormatPNG},
		{"path with dirs jpg", "results/my-chart.jpeg", FormatJPG},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			fmt, err := FormatFromPath(tc.path)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(fmt).To(Equal(tc.expected))
		})
	}
}

func TestFormatFromPath_Errors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		path        string
		errContains string
	}{
		{"no extension", "output", "no file extension"},
		{"unsupported bmp", "output.bmp", "unsupported image format"},
		{"unsupported gif", "output.gif", "unsupported image format"},
		{"unsupported tiff", "output.tiff", "unsupported image format"},
		{"unsupported webp", "output.webp", "unsupported image format"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			_, err := FormatFromPath(tc.path)
			g.Expect(err).To(HaveOccurred())
			g.Expect(err.Error()).To(ContainSubstring(tc.errContains))
		})
	}
}
