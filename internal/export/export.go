// Package export serializes the model tree and computed metrics to JSON or YAML.
package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/rotisserie/eris"
	"go.yaml.in/yaml/v3"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
)

// ExportData represents the complete model tree with all computed metrics,
// ready for serialization.
type ExportData struct {
	Root *DirectoryExport `json:"root" yaml:"root"`
}

// DirectoryExport represents a directory node with its files,
// subdirectories, and metrics.
type DirectoryExport struct {
	Name            string             `json:"name"                        yaml:"name"`
	Path            string             `json:"path"                        yaml:"path"`
	Files           []*FileExport      `json:"files,omitempty"             yaml:"files,omitempty"`
	Directories     []*DirectoryExport `json:"directories,omitempty"       yaml:"directories,omitempty"`
	Quantities      map[string]int64   `json:"quantities,omitempty"        yaml:"quantities,omitempty"`
	Measures        map[string]float64 `json:"measures,omitempty"          yaml:"measures,omitempty"`
	Classifications map[string]string  `json:"classifications,omitempty"   yaml:"classifications,omitempty"`
}

// FileExport represents a file node with its metrics.
type FileExport struct {
	Name            string             `json:"name"                        yaml:"name"`
	Path            string             `json:"path"                        yaml:"path"`
	Extension       string             `json:"extension"                   yaml:"extension"`
	IsBinary        bool               `json:"isBinary"                    yaml:"isBinary"`
	Quantities      map[string]int64   `json:"quantities,omitempty"        yaml:"quantities,omitempty"`
	Measures        map[string]float64 `json:"measures,omitempty"          yaml:"measures,omitempty"`
	Classifications map[string]string  `json:"classifications,omitempty"   yaml:"classifications,omitempty"`
}

// Export serializes the model tree and computed metrics to a file.
// Format is inferred from the file extension (.json or .yaml/.yml).
// Only metrics in the requested list are included in the output.
// If outputPath is empty, Export is a no-op and returns nil.
func Export(
	root *model.Directory,
	requested []metric.Name,
	outputPath string,
) error {
	if outputPath == "" {
		return nil
	}

	format, err := formatFromPath(outputPath)
	if err != nil {
		return err
	}

	data := ExportData{
		Root: exportDirectory(root, requested),
	}

	content, err := marshalExport(data, format)
	if err != nil {
		return err
	}

	if err := os.WriteFile(outputPath, content, 0o600); err != nil {
		return eris.Wrap(err, "failed to write export file")
	}

	return nil
}

// marshalExport serializes the export data in the specified format.
func marshalExport(data ExportData, format exportFormat) ([]byte, error) {
	switch format {
	case formatJSON:
		content, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return nil, eris.Wrap(err, "failed to marshal JSON")
		}

		// Append trailing newline for POSIX compliance.
		return append(content, '\n'), nil
	case formatYAML:
		content, err := yaml.Marshal(data)
		if err != nil {
			return nil, eris.Wrap(err, "failed to marshal YAML")
		}

		return content, nil
	default:
		return nil, eris.Errorf("unsupported export format: %d", format)
	}
}

// exportFormat represents a supported export file format.
type exportFormat int

const (
	formatJSON exportFormat = iota
	formatYAML
)

// formatFromPath infers the export format from the output file extension.
func formatFromPath(path string) (exportFormat, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".json":
		return formatJSON, nil
	case ".yaml", ".yml":
		return formatYAML, nil
	case "":
		return 0, eris.New("output path has no file extension; supported formats: json, yaml, yml")
	default:
		return 0, eris.Errorf("unsupported export format %q; supported formats: json, yaml, yml", ext)
	}
}

// exportDirectory recursively converts a model.Directory into a DirectoryExport.
func exportDirectory(dir *model.Directory, requested []metric.Name) *DirectoryExport {
	de := &DirectoryExport{
		Name: dir.Name,
		Path: dir.Path,
	}

	collectDirectoryMetrics(de, dir, requested)

	for _, f := range dir.Files {
		de.Files = append(de.Files, exportFile(f, requested))
	}

	for _, sub := range dir.Dirs {
		de.Directories = append(de.Directories, exportDirectory(sub, requested))
	}

	return de
}

// exportFile converts a model.File into a FileExport.
func exportFile(f *model.File, requested []metric.Name) *FileExport {
	fe := &FileExport{
		Name:      f.Name,
		Path:      f.Path,
		Extension: f.Extension,
		IsBinary:  f.IsBinary,
	}

	collectFileMetrics(fe, f, requested)

	return fe
}

// collectDirectoryMetrics populates a DirectoryExport's metric maps from the
// model directory, including only the requested metrics that are present.
func collectDirectoryMetrics(
	de *DirectoryExport,
	dir *model.Directory,
	requested []metric.Name,
) {
	for _, name := range requested {
		if q, ok := dir.Quantity(name); ok {
			de.Quantities = addQuantity(de.Quantities, string(name), q)
		}

		if m, ok := dir.Measure(name); ok {
			de.Measures = addMeasure(de.Measures, string(name), m)
		}

		if c, ok := dir.Classification(name); ok {
			de.Classifications = addClassification(de.Classifications, string(name), c)
		}
	}
}

// collectFileMetrics populates a FileExport's metric maps from the model file,
// including only the requested metrics that are present.
func collectFileMetrics(
	fe *FileExport,
	f *model.File,
	requested []metric.Name,
) {
	for _, name := range requested {
		if q, ok := f.Quantity(name); ok {
			fe.Quantities = addQuantity(fe.Quantities, string(name), q)
		}

		if m, ok := f.Measure(name); ok {
			fe.Measures = addMeasure(fe.Measures, string(name), m)
		}

		if c, ok := f.Classification(name); ok {
			fe.Classifications = addClassification(fe.Classifications, string(name), c)
		}
	}
}

func addQuantity(m map[string]int64, key string, value int64) map[string]int64 {
	if m == nil {
		m = make(map[string]int64)
	}

	m[key] = value

	return m
}

func addMeasure(m map[string]float64, key string, value float64) map[string]float64 {
	if m == nil {
		m = make(map[string]float64)
	}

	m[key] = value

	return m
}

func addClassification(m map[string]string, key, value string) map[string]string {
	if m == nil {
		m = make(map[string]string)
	}

	m[key] = value

	return m
}
