package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"go.yaml.in/yaml/v3"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
)

// Metric name constants used across tests.
const (
	fileSize   metric.Name = "file-size"
	lineCount  metric.Name = "line-count"
	complexity metric.Name = "complexity"
	fileType   metric.Name = "file-type"
)

// makeFile creates a *model.File with the given name and extension.
func makeFile(name, ext string, binary bool) *model.File {
	return &model.File{
		Path:      name,
		Name:      name,
		Extension: ext,
		IsBinary:  binary,
	}
}

// sampleTree builds a small model tree for use in tests:
//
//	root/
//	  main.go   (file-size=120, file-type="go")
//	  README.md (file-size=45,  file-type="md")
func sampleTree() *model.Directory {
	goFile := makeFile("main.go", "go", false)
	goFile.SetQuantity(fileSize, 120)
	goFile.SetClassification(fileType, "go")

	mdFile := makeFile("README.md", "md", false)
	mdFile.SetQuantity(fileSize, 45)
	mdFile.SetClassification(fileType, "md")

	root := &model.Directory{
		Name: "root",
		Path: "root",
		Files: []*model.File{
			goFile,
			mdFile,
		},
	}
	root.SetQuantity(fileSize, 165)

	return root
}

// TestExport_JSON exports a simple tree to JSON and verifies the output
// is valid JSON with the expected structure.
func TestExport_JSON(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := sampleTree()
	out := filepath.Join(t.TempDir(), "export.json")

	err := Export(root, []metric.Name{fileSize, fileType}, out)
	g.Expect(err).NotTo(HaveOccurred())

	raw, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	var data ExportData
	err = json.Unmarshal(raw, &data)
	g.Expect(err).NotTo(HaveOccurred(), "output must be valid JSON")

	g.Expect(data.Root).NotTo(BeNil())
	g.Expect(data.Root.Name).To(Equal("root"))
	g.Expect(data.Root.Files).To(HaveLen(2))
	g.Expect(data.Root.Quantities).To(HaveKeyWithValue("file-size", int64(165)))
}

// TestExport_YAML exports to a .yaml file and verifies valid YAML output.
func TestExport_YAML(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := sampleTree()
	out := filepath.Join(t.TempDir(), "export.yaml")

	err := Export(root, []metric.Name{fileSize, fileType}, out)
	g.Expect(err).NotTo(HaveOccurred())

	raw, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	var data ExportData
	err = yaml.Unmarshal(raw, &data)
	g.Expect(err).NotTo(HaveOccurred(), "output must be valid YAML")

	g.Expect(data.Root).NotTo(BeNil())
	g.Expect(data.Root.Name).To(Equal("root"))
	g.Expect(data.Root.Files).To(HaveLen(2))
}

// TestExport_YML verifies that .yml is accepted as a YAML extension.
func TestExport_YML(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := sampleTree()
	out := filepath.Join(t.TempDir(), "export.yml")

	err := Export(root, []metric.Name{fileSize}, out)
	g.Expect(err).NotTo(HaveOccurred())

	raw, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	var data ExportData
	err = yaml.Unmarshal(raw, &data)
	g.Expect(err).NotTo(HaveOccurred(), ".yml must produce valid YAML")

	g.Expect(data.Root).NotTo(BeNil())
	g.Expect(data.Root.Name).To(Equal("root"))
}

// TestExport_UnsupportedFormat verifies that an unsupported extension
// returns an error.
func TestExport_UnsupportedFormat(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := sampleTree()
	out := filepath.Join(t.TempDir(), "export.txt")

	err := Export(root, []metric.Name{fileSize}, out)
	g.Expect(err).To(HaveOccurred())
}

// TestExport_MetricFiltering sets multiple metrics on files but requests
// only a subset. Only requested metrics should appear in the output.
func TestExport_MetricFiltering(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := makeFile("app.go", "go", false)
	f.SetQuantity(fileSize, 200)
	f.SetQuantity(lineCount, 50)
	f.SetClassification(fileType, "go")

	root := &model.Directory{
		Name:  "project",
		Path:  "project",
		Files: []*model.File{f},
	}
	root.SetQuantity(fileSize, 200)
	root.SetQuantity(lineCount, 50)

	out := filepath.Join(t.TempDir(), "filtered.json")

	// Request only fileSize — lineCount and fileType should be absent.
	err := Export(root, []metric.Name{fileSize}, out)
	g.Expect(err).NotTo(HaveOccurred())

	raw, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	var data ExportData
	err = json.Unmarshal(raw, &data)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(data.Root).NotTo(BeNil())

	if data.Root == nil {
		return
	}

	// Root directory should have fileSize but not lineCount.
	g.Expect(data.Root.Quantities).To(HaveKeyWithValue("file-size", int64(200)))
	g.Expect(data.Root.Quantities).NotTo(HaveKey("line-count"))

	g.Expect(data.Root.Files).To(HaveLen(1))

	if len(data.Root.Files) == 0 {
		return
	}

	fe := data.Root.Files[0]

	// File should have fileSize but not lineCount or fileType.
	g.Expect(fe.Quantities).To(HaveKeyWithValue("file-size", int64(200)))
	g.Expect(fe.Quantities).NotTo(HaveKey("line-count"))
	g.Expect(fe.Classifications).NotTo(HaveKey("file-type"))
}

// TestExport_EmptyDirectory exports an empty directory (no files,
// no subdirectories) and verifies valid output.
func TestExport_EmptyDirectory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "empty",
		Path: "empty",
	}

	out := filepath.Join(t.TempDir(), "empty.json")

	err := Export(root, []metric.Name{fileSize}, out)
	g.Expect(err).NotTo(HaveOccurred())

	raw, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	var data ExportData
	err = json.Unmarshal(raw, &data)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(data.Root).NotTo(BeNil())
	g.Expect(data.Root.Name).To(Equal("empty"))
	g.Expect(data.Root.Files).To(BeEmpty())
	g.Expect(data.Root.Directories).To(BeEmpty())
}

// TestExport_NestedDirectories exports a 3-level deep tree and verifies
// the hierarchy is preserved.
func TestExport_NestedDirectories(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	leaf := makeFile("util.go", "go", false)
	leaf.SetQuantity(fileSize, 30)

	deepDir := &model.Directory{
		Name:  "deep",
		Path:  "root/mid/deep",
		Files: []*model.File{leaf},
	}

	midDir := &model.Directory{
		Name: "mid",
		Path: "root/mid",
		Dirs: []*model.Directory{deepDir},
	}

	root := &model.Directory{
		Name: "root",
		Path: "root",
		Dirs: []*model.Directory{midDir},
	}

	out := filepath.Join(t.TempDir(), "nested.json")

	err := Export(root, []metric.Name{fileSize}, out)
	g.Expect(err).NotTo(HaveOccurred())

	raw, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	var data ExportData

	err = json.Unmarshal(raw, &data)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(data.Root).NotTo(BeNil())

	if data.Root == nil {
		return
	}

	// Level 1: root has one subdirectory.
	g.Expect(data.Root.Name).To(Equal("root"))
	g.Expect(data.Root.Directories).To(HaveLen(1))

	if len(data.Root.Directories) == 0 {
		return
	}

	// Level 2: mid.
	mid := data.Root.Directories[0]
	g.Expect(mid.Name).To(Equal("mid"))
	g.Expect(mid.Directories).To(HaveLen(1))

	if len(mid.Directories) == 0 {
		return
	}

	// Level 3: deep contains one file.
	deep := mid.Directories[0]
	g.Expect(deep.Name).To(Equal("deep"))
	g.Expect(deep.Files).To(HaveLen(1))
}

// TestExport_BinaryFileFlag verifies that the isBinary flag is correctly
// serialized for binary files.
func TestExport_BinaryFileFlag(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	binFile := makeFile("image.png", "png", true)
	binFile.SetQuantity(fileSize, 5000)

	textFile := makeFile("notes.txt", "txt", false)
	textFile.SetQuantity(fileSize, 100)

	root := &model.Directory{
		Name: "mixed",
		Path: "mixed",
		Files: []*model.File{
			binFile,
			textFile,
		},
	}

	out := filepath.Join(t.TempDir(), "binary.json")

	err := Export(root, []metric.Name{fileSize}, out)
	g.Expect(err).NotTo(HaveOccurred())

	raw, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	var data ExportData

	err = json.Unmarshal(raw, &data)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(data.Root).NotTo(BeNil())

	if data.Root == nil {
		return
	}

	g.Expect(data.Root.Files).To(HaveLen(2))

	if len(data.Root.Files) < 2 {
		return
	}

	// Find the binary and text files (order may vary).
	var bin, txt *FileExport

	for _, fe := range data.Root.Files {
		if fe == nil {
			continue
		}

		switch fe.Name {
		case "image.png":
			bin = fe
		case "notes.txt":
			txt = fe
		default:
			// ignore other files
		}
	}

	g.Expect(bin).NotTo(BeNil(), "binary file should be present")
	g.Expect(txt).NotTo(BeNil(), "text file should be present")

	if bin == nil || txt == nil {
		return
	}

	g.Expect(bin.IsBinary).To(BeTrue())
	g.Expect(txt.IsBinary).To(BeFalse())
}

// TestExport_AllMetricTypes sets quantity, measure, AND classification
// metrics on a file and verifies all three map types are populated.
func TestExport_AllMetricTypes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := makeFile("rich.go", "go", false)
	f.SetQuantity(fileSize, 500)
	f.SetMeasure(complexity, 7.25)
	f.SetClassification(fileType, "go")

	root := &model.Directory{
		Name:  "metrics",
		Path:  "metrics",
		Files: []*model.File{f},
	}
	root.SetQuantity(fileSize, 500)
	root.SetMeasure(complexity, 7.25)
	root.SetClassification(fileType, "source")

	out := filepath.Join(t.TempDir(), "all-metrics.json")

	err := Export(root, []metric.Name{fileSize, complexity, fileType}, out)
	g.Expect(err).NotTo(HaveOccurred())

	raw, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	var data ExportData

	err = json.Unmarshal(raw, &data)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(data.Root).NotTo(BeNil())

	if data.Root == nil {
		return
	}

	// Directory-level metrics.
	g.Expect(data.Root.Quantities).To(HaveKeyWithValue("file-size", int64(500)))
	g.Expect(data.Root.Measures).To(HaveKeyWithValue("complexity", 7.25))
	g.Expect(data.Root.Classifications).To(HaveKeyWithValue("file-type", Equal("source")))

	g.Expect(data.Root.Files).To(HaveLen(1))

	if len(data.Root.Files) == 0 {
		return
	}

	fe := data.Root.Files[0]

	// File-level metrics.
	g.Expect(fe.Quantities).To(HaveKeyWithValue("file-size", int64(500)))
	g.Expect(fe.Measures).To(HaveKeyWithValue("complexity", 7.25))
	g.Expect(fe.Classifications).To(HaveKeyWithValue("file-type", Equal("go")))
}
