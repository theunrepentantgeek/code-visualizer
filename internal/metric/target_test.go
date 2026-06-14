package metric

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestTargetString(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(File.String()).To(Equal("file"))
	g.Expect(Directory.String()).To(Equal("directory"))
}

func TestTargetStringUnknown(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(Target(99).String()).To(Equal("unknown"))
}
