package metric

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/scan"
)

func TestKindConstants(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(Quantity).To(Equal(Kind(0)))
	g.Expect(Measure).To(Equal(Kind(1)))
	g.Expect(Classification).To(Equal(Kind(2)))
}

func TestMetricName_IsValid(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	valid := []MetricName{FileSize, FileLines, FileType, FileAge, FileFreshness, AuthorCount}
	for _, m := range valid {
		g.Expect(m.IsValid()).To(BeTrue(), "expected %q to be valid", m)
	}

	g.Expect(MetricName("unknown").IsValid()).To(BeFalse())
	g.Expect(MetricName("").IsValid()).To(BeFalse())
	g.Expect(MetricName("FILE-SIZE").IsValid()).To(BeFalse())
}

func TestMetricName_IsNumeric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	numeric := []MetricName{FileSize, FileLines, FileAge, FileFreshness, AuthorCount}
	for _, m := range numeric {
		g.Expect(m.IsNumeric()).To(BeTrue(), "expected %q to be numeric", m)
	}

	g.Expect(FileType.IsNumeric()).To(BeFalse())
}

func TestMetricName_IsGitRequired(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	gitRequired := []MetricName{FileAge, FileFreshness, AuthorCount}
	for _, m := range gitRequired {
		g.Expect(m.IsGitRequired()).To(BeTrue(), "expected %q to be git-required", m)
	}

	nonGit := []MetricName{FileSize, FileLines, FileType}
	for _, m := range nonGit {
		g.Expect(m.IsGitRequired()).To(BeFalse(), "expected %q to NOT be git-required", m)
	}
}

func TestExtractFileSize_RegularFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	node := scan.FileNode{Size: 4096}
	val := ExtractFileSize(node)
	g.Expect(val).To(Equal(float64(4096)))
}

func TestExtractFileSize_ZeroByteFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	node := scan.FileNode{Size: 0}
	val := ExtractFileSize(node)
	g.Expect(val).To(Equal(float64(0)))
}

func TestExtractFileSize_LargeFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	node := scan.FileNode{Size: 1_073_741_824} // 1 GiB
	val := ExtractFileSize(node)
	g.Expect(val).To(Equal(float64(1_073_741_824)))
}

func TestExtractFileLines_TextFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	node := scan.FileNode{LineCount: 42}
	val := ExtractFileLines(node)
	g.Expect(val).To(Equal(float64(42)))
}

func TestExtractFileLines_EmptyFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	node := scan.FileNode{LineCount: 0}
	val := ExtractFileLines(node)
	g.Expect(val).To(Equal(float64(0)))
}

func TestExtractFileType(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(ExtractFileType(scan.FileNode{FileType: "go"})).To(Equal("go"))
	g.Expect(ExtractFileType(scan.FileNode{FileType: "no-extension"})).To(Equal("no-extension"))
}

func TestExtractFileType_Extension(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Extension-based: .go → "go", .tar.gz → "gz", no ext → "no-extension"
	g.Expect(ExtractFileType(scan.FileNode{FileType: "gz"})).To(Equal("gz"))
}
