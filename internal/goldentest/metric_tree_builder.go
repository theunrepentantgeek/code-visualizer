package goldentest

import (
	"hash/fnv"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// classificationValues is a small fixed vocabulary used for synthetic
// classification base values, chosen deterministically by hash.
var classificationValues = []string{"alpha", "beta", "gamma", "delta"}

const (
	rootDirName = "root"
	visPublic   = "public"
	visPrivate  = "private"
	goExtension = "go"
)

// synthInt returns a deterministic int64 in [1, 1000] derived from a seed.
func synthInt(seed string) int64 {
	return int64(hashOf(seed)%1000) + 1
}

// synthFloat returns a deterministic float64 in [0, 100) derived from a seed.
func synthFloat(seed string) float64 {
	return float64(hashOf(seed)%10000) / 100.0
}

// synthClass returns a deterministic classification value derived from a seed.
func synthClass(seed string) string {
	return classificationValues[hashOf(seed)%uint64(len(classificationValues))]
}

func hashOf(seed string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(seed))

	return h.Sum64()
}

// baseMetricSetter is the subset of model.MetricContainer used to populate
// synthetic base values. *model.File, *model.Declaration and *model.Commit all
// embed model.MetricContainer, so they satisfy this interface.
type baseMetricSetter interface {
	SetQuantity(name metric.Name, value int64)
	SetMeasure(name metric.Name, value float64)
	SetClassification(name metric.Name, value string)
}

// setBaseMetric writes a deterministic synthetic value for desc onto the
// container, keyed by the descriptor's kind. nodeID makes the value unique per
// node so aggregation produces non-trivial results.
func setBaseMetric(mc baseMetricSetter, desc provider.BaseMetricDescriptor, nodeID string) {
	seed := string(desc.Name) + "|" + nodeID
	switch desc.Kind {
	case metric.Quantity:
		mc.SetQuantity(desc.Name, synthInt(seed))
	case metric.Measure:
		mc.SetMeasure(desc.Name, synthFloat(seed))
	case metric.Classification:
		mc.SetClassification(desc.Name, synthClass(seed))
	default:
	}

	// Metrics that declare filters also need filter.base values so filtered
	// aggregation has data to read.
	for _, fn := range desc.Filters {
		filtered := metric.MetricExpression{Filter: fn, Base: desc.Name}.ResultName()
		switch desc.Kind {
		case metric.Quantity:
			mc.SetQuantity(filtered, synthInt(seed+"|"+string(fn)))
		case metric.Measure:
			mc.SetMeasure(filtered, synthFloat(seed+"|"+string(fn)))
		case metric.Classification:
			mc.SetClassification(filtered, synthClass(seed+"|"+string(fn)))
		default:
		}
	}
}

// declarationSpecs gives a representative spread covering both visibilities and
// several kinds so declaration filters and kind-matching are exercised.
var declarationSpecs = []struct {
	name       string
	kind       string
	visibility string
}{
	{"PublicType", model.DeclKindType, visPublic},
	{"privateType", model.DeclKindType, visPrivate},
	{"PublicFunc", model.DeclKindFunction, visPublic},
	{"privateFunc", model.DeclKindFunction, visPrivate},
	{"PublicMethod", model.DeclKindMethod, visPublic},
	{"privateConst", model.DeclKindConstant, visPrivate},
}

// newDeclarations builds a fixed set of declarations for a file, each carrying
// every declaration-level base metric.
func newDeclarations(fileID string, declLevel []provider.BaseMetricDescriptor) []*model.Declaration {
	decls := make([]*model.Declaration, 0, len(declarationSpecs))
	for _, ds := range declarationSpecs {
		d := &model.Declaration{Name: ds.name, Kind: ds.kind, Visibility: ds.visibility}
		for _, desc := range declLevel {
			setBaseMetric(d, desc, fileID+"/"+ds.name)
		}

		decls = append(decls, d)
	}

	return decls
}

// newCommits builds a fixed set of commits for a file, each carrying every
// commit-level base metric.
func newCommits(fileID string, commitLevel []provider.BaseMetricDescriptor) []*model.Commit {
	commits := make([]*model.Commit, 0, 2)

	for i := range 2 {
		suffix := string(rune('0' + i))

		c := &model.Commit{Hash: fileID + "-commit-" + suffix}
		for _, desc := range commitLevel {
			setBaseMetric(c, desc, fileID+"/commit/"+suffix)
		}

		commits = append(commits, c)
	}

	return commits
}

// newMetricFile builds a file populated with all file-level base metrics plus
// declarations and commits carrying their level's base metrics.
func newMetricFile(path, name, ext string,
	fileLevel, declLevel, commitLevel []provider.BaseMetricDescriptor,
) *model.File {
	f := &model.File{Path: path, Name: name, Extension: ext}
	for _, desc := range fileLevel {
		setBaseMetric(f, desc, path)
	}

	f.Declarations = newDeclarations(path, declLevel)
	f.Commits = newCommits(path, commitLevel)

	return f
}

// buildMetricTree returns a fixed nested directory tree where every node level
// carries deterministic synthetic values for every base metric in the registry.
func buildMetricTree() *model.Directory {
	fileLevel := provider.AllBaseForLevel(metric.LevelFile)
	declLevel := provider.AllBaseForLevel(metric.LevelDeclaration)
	commitLevel := provider.AllBaseForLevel(metric.LevelCommit)

	mk := func(path, name string) *model.File {
		return newMetricFile(path, name, goExtension, fileLevel, declLevel, commitLevel)
	}

	return &model.Directory{
		Path: rootDirName,
		Name: rootDirName,
		Files: []*model.File{
			mk("root/a.go", "a.go"),
			mk("root/b.go", "b.go"),
		},
		Dirs: []*model.Directory{
			{
				Path:  "root/sub",
				Name:  "sub",
				Files: []*model.File{mk("root/sub/c.go", "c.go")},
				Dirs: []*model.Directory{
					{
						Path:  "root/sub/deep",
						Name:  "deep",
						Files: []*model.File{mk("root/sub/deep/d.go", "d.go")},
					},
				},
			},
		},
	}
}
