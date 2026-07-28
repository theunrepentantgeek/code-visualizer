package viz

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestGrainConstants(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(string(GrainFile)).To(Equal("file"))
	g.Expect(string(GrainDirectory)).To(Equal("directory"))
}
