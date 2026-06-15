package metric

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestMetricLevel_String(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(LevelFile.String()).To(Equal("file"))
	g.Expect(LevelDeclaration.String()).To(Equal("declaration"))
	g.Expect(LevelCommit.String()).To(Equal("commit"))
	g.Expect(LevelDirectory.String()).To(Equal("directory"))
}

func TestMetricLevel_UnknownString(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(MetricLevel(99).String()).To(Equal("unknown"))
}
