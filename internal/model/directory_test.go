package model

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestDirectorySetAndGetQuantity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	d := &Directory{}

	d.SetQuantity("folder-size", 9999)

	v, ok := d.Quantity("folder-size")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(int64(9999)))
}

func TestDirectoryGetUnsetMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	d := &Directory{}

	_, ok := d.Quantity("unset")
	g.Expect(ok).To(BeFalse())
}

func TestDirectorySetAndGetMeasure(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	d := &Directory{}

	d.SetMeasure("complexity", 2.718)

	v, ok := d.Measure("complexity")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(2.718))
}

func TestDirectorySetAndGetClassification(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	d := &Directory{}

	d.SetClassification("file-type", "go")

	v, ok := d.Classification("file-type")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal("go"))
}

func TestDirectoryGetUnsetMeasureAndClassification(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	d := &Directory{}

	_, ok := d.Measure("unset")
	g.Expect(ok).To(BeFalse())

	_, ok = d.Classification("unset")
	g.Expect(ok).To(BeFalse())
}

func TestDirectoryPointerSlices(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	child := &File{Path: "/src/a.go", Name: "a.go"}
	subdir := &Directory{Path: "/src/sub", Name: "sub"}
	d := &Directory{
		Path:  "/src",
		Name:  "src",
		Files: []*File{child},
		Dirs:  []*Directory{subdir},
	}

	g.Expect(d.Path).To(Equal("/src"))
	g.Expect(d.Name).To(Equal("src"))
	g.Expect(d.Files).To(HaveLen(1))
	g.Expect(d.Dirs).To(HaveLen(1))
	g.Expect(d.Files[0].Name).To(Equal("a.go"))
	g.Expect(d.Dirs[0].Name).To(Equal("sub"))
}
