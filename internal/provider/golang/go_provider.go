package golang

import (
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/rotisserie/eris"
	"golang.org/x/sync/singleflight"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// goExtractor extracts one metric value from fileStats and sets it on the model file.
type goExtractor func(name metric.Name, stats *fileStats, f *model.File)

// goProvider is a data-driven implementation of provider.Interface for Go metrics.
type goProvider struct {
	name           metric.Name
	kind           metric.Kind
	description    string
	defaultPalette palette.PaletteName
	extract        goExtractor
	onFile         func()
}

func (p *goProvider) Name() metric.Name                   { return p.name }
func (p *goProvider) Kind() metric.Kind                   { return p.kind }
func (p *goProvider) Description() string                 { return p.description }
func (*goProvider) Dependencies() []metric.Name           { return nil }
func (p *goProvider) DefaultPalette() palette.PaletteName { return p.defaultPalette }
func (p *goProvider) SetOnFileProcessed(fn func())        { p.onFile = fn }

func (p *goProvider) Load(root *model.Directory) error {
	walkGoFiles(root, p.name, p.onFile, p.extract)

	return nil
}

// providerDef holds the static fields for one goProvider.
type providerDef struct {
	kind           metric.Kind
	description    string
	defaultPalette palette.PaletteName
	extract        goExtractor
}

// newProvider creates a fresh goProvider for the given metric name.
func newProvider(name metric.Name) *goProvider {
	def, ok := providerDefs[name]
	if !ok {
		panic("newProvider: unknown Go metric name: " + string(name))
	}

	return &goProvider{
		name:           name,
		kind:           def.kind,
		description:    def.description,
		defaultPalette: def.defaultPalette,
		extract:        def.extract,
	}
}

// statsCache caches parsed fileStats per file path.
type statsCache struct {
	mu    sync.Mutex
	group singleflight.Group
	stats map[string]*fileStats
}

var globalCache = &statsCache{
	stats: make(map[string]*fileStats),
}

var globalModuleCache = newModuleCache()

// getOrAnalyze returns the cached fileStats for path, parsing if necessary.
// Concurrent requests for the same path are deduplicated via singleflight.
func getOrAnalyze(path string) (*fileStats, error) {
	globalCache.mu.Lock()
	if s, ok := globalCache.stats[path]; ok {
		globalCache.mu.Unlock()

		return s, nil
	}
	globalCache.mu.Unlock()

	result, err, _ := globalCache.group.Do(path, func() (any, error) {
		dir := filepath.Dir(path)
		modulePath := globalModuleCache.findModulePath(dir)

		s, err := analyzeFile(path, modulePath)
		if err != nil {
			return nil, err
		}

		globalCache.mu.Lock()
		globalCache.stats[path] = s
		globalCache.mu.Unlock()

		return s, nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "analyzing Go file")
	}

	stats, ok := result.(*fileStats)
	if !ok {
		return nil, eris.New("unexpected type from singleflight result")
	}

	return stats, nil
}

// walkGoFiles walks all .go files under root and calls the extract function
// with cached fileStats for each. Non-.go files are silently skipped.
func walkGoFiles(
	root *model.Directory,
	name metric.Name,
	onFile func(),
	extract goExtractor,
) {
	model.WalkFiles(root, func(f *model.File) {
		if onFile != nil {
			defer onFile()
		}

		if f.Extension != "go" {
			return
		}

		stats, err := getOrAnalyze(f.Path)
		if err != nil {
			slog.Warn("could not analyze Go file", "path", f.Path, "error", err)

			return
		}

		extract(name, stats, f)
	})
}

// quantityField returns a goExtractor that reads an int64 field from fileStats.
func quantityField(fn func(*fileStats) int64) goExtractor {
	return func(name metric.Name, stats *fileStats, f *model.File) {
		f.SetQuantity(name, fn(stats))
	}
}

// measureField returns a goExtractor that reads a float64 field from fileStats.
func measureField(fn func(*fileStats) float64) goExtractor {
	return func(name metric.Name, stats *fileStats, f *model.File) {
		f.SetMeasure(name, fn(stats))
	}
}

// ResetCacheForTesting clears the global caches. Test use only.
func ResetCacheForTesting() {
	globalCache = &statsCache{
		stats: make(map[string]*fileStats),
	}
	globalModuleCache = newModuleCache()
}
