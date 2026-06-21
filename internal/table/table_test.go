package table

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

func TestNew_ReturnsNonNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tbl := New("Name", "Value")

	g.Expect(tbl).NotTo(BeNil())
}

func TestNew_HeaderRowIsFirst(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tbl := New("Col1", "Col2")

	g.Expect(tbl.content).To(HaveLen(1))
	g.Expect(tbl.content[0]).To(Equal([]string{"Col1", "Col2"}))
}

func TestAddRow_AddsRow(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tbl := New("Name", "Value")
	tbl.AddRow("alpha", "1")
	tbl.AddRow("beta", "2")

	g.Expect(tbl.content).To(HaveLen(3)) // header + 2 rows
}

func TestAddRow_TracksColumnWidths(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tbl := New("A", "B")
	tbl.AddRow("longer", "x")

	// "longer" (6) > "A" (1)
	g.Expect(tbl.widths[0]).To(Equal(6))
	// "B" (1) == "x" (1)
	g.Expect(tbl.widths[1]).To(Equal(1))
}

func TestWriteTo_OutputContainsHeader(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tbl := New("Name", "Count")

	var buf strings.Builder
	tbl.WriteTo(&buf)

	g.Expect(buf.String()).To(ContainSubstring("Name"))
	g.Expect(buf.String()).To(ContainSubstring("Count"))
}

func TestWriteTo_OutputContainsDivider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tbl := New("Name", "Count")
	tbl.AddRow("foo", "1")

	var buf strings.Builder
	tbl.WriteTo(&buf)

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	g.Expect(lines).To(HaveLen(3)) // header, divider, data row

	// divider line starts with |--- pattern
	g.Expect(lines[1]).To(MatchRegexp(`^\|[-]+\|`))
}

func TestWriteTo_OutputContainsDataRows(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tbl := New("Metric", "Value")
	tbl.AddRow("file-size", "bytes")
	tbl.AddRow("file-lines", "count")

	var buf strings.Builder
	tbl.WriteTo(&buf)

	output := buf.String()
	g.Expect(output).To(ContainSubstring("file-size"))
	g.Expect(output).To(ContainSubstring("file-lines"))
	g.Expect(output).To(ContainSubstring("bytes"))
	g.Expect(output).To(ContainSubstring("count"))
}

func TestWriteTo_ColumnsAlignedByMaxWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// "short" and "a very long value" — widths should be padded to the max
	tbl := New("Key", "Description")
	tbl.AddRow("a", "short")
	tbl.AddRow("b", "a very long value")

	var buf strings.Builder
	tbl.WriteTo(&buf)

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	// All data lines should be the same length (consistent column alignment)
	g.Expect(lines[0]).To(HaveLen(len(lines[2]))) // header == first data row width
	g.Expect(lines[2]).To(HaveLen(len(lines[3]))) // first data row == second data row
}

func TestWriteTo_EmptyTable(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// A table with only a header should produce header + divider
	tbl := New("Name")

	var buf strings.Builder
	tbl.WriteTo(&buf)

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	g.Expect(lines).To(HaveLen(2)) // header + divider only
}

func TestWriteTo_SingleColumn(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tbl := New("Items")
	tbl.AddRow("alpha")
	tbl.AddRow("beta")

	var buf strings.Builder
	tbl.WriteTo(&buf)

	output := buf.String()
	g.Expect(output).To(ContainSubstring("| Items |"))
	g.Expect(output).To(ContainSubstring("| alpha |"))
	g.Expect(output).To(ContainSubstring("| beta  |"))
}

func TestWriteTo_PipeDelimited(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tbl := New("A", "B")
	tbl.AddRow("x", "y")

	var buf strings.Builder
	tbl.WriteTo(&buf)

	// Every line should start and end with |
	for line := range strings.SplitSeq(strings.TrimRight(buf.String(), "\n"), "\n") {
		g.Expect(line).To(HavePrefix("|"))
		g.Expect(line).To(HaveSuffix("|"))
	}
}

func TestSetMaxWidth_NoWrapNeeded(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tbl := New("Name", "Description")
	tbl.AddRow("foo", "short text")
	tbl.SetMaxWidth(200)

	var buf strings.Builder
	tbl.WriteTo(&buf)

	// Content fits; no wrapping should occur — all lines have the same length.
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	g.Expect(lines).To(HaveLen(3))
	g.Expect(lines[0]).To(HaveLen(len(lines[2])))
}

func TestSetMaxWidth_WrapsLastColumn(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tbl := New("Name", "Description")
	// Long description that would exceed a narrow max width.
	tbl.AddRow("foo", "this is a very long description that should be wrapped to fit the console width")
	tbl.SetMaxWidth(40)

	var buf strings.Builder
	tbl.WriteTo(&buf)

	// The output must contain the full text across multiple lines.
	output := buf.String()
	g.Expect(output).To(ContainSubstring("this is a very long"))
	g.Expect(output).To(ContainSubstring("width"))

	// All lines must fit within the max width.
	for line := range strings.SplitSeq(strings.TrimRight(output, "\n"), "\n") {
		g.Expect(len(line)).To(BeNumerically("<=", 40), "line too long: %q", line)
	}
}

func TestSetMaxWidth_AllLinesStartAndEndWithPipe(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tbl := New("Col", "Description")
	tbl.AddRow("x", "word one two three four five six seven eight nine ten")
	tbl.SetMaxWidth(35)

	var buf strings.Builder
	tbl.WriteTo(&buf)

	for line := range strings.SplitSeq(strings.TrimRight(buf.String(), "\n"), "\n") {
		g.Expect(line).To(HavePrefix("|"))
		g.Expect(line).To(HaveSuffix("|"))
	}
}

func TestWordWrap_ShortText(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	result := wordWrap("hello", 20)

	g.Expect(result).To(Equal([]string{"hello"}))
}

func TestWordWrap_LongText(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	result := wordWrap("one two three four five", 12)

	g.Expect(result).To(HaveLen(3))
	g.Expect(result[0]).To(Equal("one two"))
	g.Expect(result[1]).To(Equal("three four"))
	g.Expect(result[2]).To(Equal("five"))
}

func TestWordWrap_EmptyString(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	result := wordWrap("", 20)

	g.Expect(result).To(Equal([]string{""}))
}
