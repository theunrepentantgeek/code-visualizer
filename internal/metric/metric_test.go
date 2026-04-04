package metric

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/scan"
)

func TestMetricName_IsValid(t *testing.T) {
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
	g := NewGomegaWithT(t)

	numeric := []MetricName{FileSize, FileLines, FileAge, FileFreshness, AuthorCount}
	for _, m := range numeric {
		g.Expect(m.IsNumeric()).To(BeTrue(), "expected %q to be numeric", m)
	}

	g.Expect(FileType.IsNumeric()).To(BeFalse())
}

func TestMetricName_IsGitRequired(t *testing.T) {
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
	g := NewGomegaWithT(t)

	node := scan.FileNode{Size: 4096}
	val := ExtractFileSize(node)
	g.Expect(val).To(Equal(float64(4096)))
}

func TestExtractFileSize_ZeroByteFile(t *testing.T) {
	g := NewGomegaWithT(t)

	node := scan.FileNode{Size: 0}
	val := ExtractFileSize(node)
	g.Expect(val).To(Equal(float64(0)))
}

func TestExtractFileSize_LargeFile(t *testing.T) {
	g := NewGomegaWithT(t)

	node := scan.FileNode{Size: 1_073_741_824} // 1 GiB
	val := ExtractFileSize(node)
	g.Expect(val).To(Equal(float64(1_073_741_824)))
}
