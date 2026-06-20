package spiral_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	pkginks "github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/spiral"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestMain(m *testing.M) {
	filesystem.Register()
	m.Run()
}

func makeFile(name, ext string, size int64) *model.File {
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
				makeFile("a.go", "go", 100),
				makeFile("b.go", "go", 200),
			},
			SizeValue: 300, FillValue: 300, FillLabel: "go",
		},
		{
			Start: t0.Add(time.Hour), End: t0.Add(2 * time.Hour),
			Files: []*model.File{
				makeFile("c.py", "py", 50),
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

func TestBuildInks_Numeric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	inks := spiral.BuildInks(
		buckets,
		stages.RequestedMetrics{},
		filesystem.FileSize,
		palette.Temperature,
		"",
		"",
	)

	g.Expect(inks.Fill.Info().Kind).To(Equal(pkginks.KindNumeric))
	g.Expect(inks.Border.Info().Kind).To(Equal(pkginks.KindFixed))
}

func TestBuildInks_Categorical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	inks := spiral.BuildInks(
		buckets,
		stages.RequestedMetrics{},
		filesystem.FileType,
		palette.Categorization,
		"",
		"",
	)

	g.Expect(inks.Fill.Info().Kind).To(Equal(pkginks.KindCategorical))
}

func TestBuildInks_NoMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	inks := spiral.BuildInks(buckets, stages.RequestedMetrics{}, "", "", "", "")

	g.Expect(inks.Fill.Info().Kind).To(Equal(pkginks.KindFixed))
	g.Expect(inks.Border.Info().Kind).To(Equal(pkginks.KindFixed))
}
