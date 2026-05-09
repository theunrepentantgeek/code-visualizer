package canvas

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestLayerOrdering_BackgroundFirst(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(LayerBackground).To(BeNumerically("<", LayerStructure))
	g.Expect(LayerStructure).To(BeNumerically("<", LayerContent))
	g.Expect(LayerContent).To(BeNumerically("<", LayerOverlay))
}

func TestLayerOrdering_GapsExist(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(LayerStructure - LayerBackground).To(Equal(Layer(10)))
	g.Expect(LayerContent - LayerStructure).To(Equal(Layer(10)))
	g.Expect(LayerOverlay - LayerContent).To(Equal(Layer(10)))
}

func TestLayer_CustomValue_BetweenStandard(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	custom := Layer(15)
	g.Expect(custom).To(BeNumerically(">", LayerStructure))
	g.Expect(custom).To(BeNumerically("<", LayerContent))
}
