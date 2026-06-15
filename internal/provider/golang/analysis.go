package golang

import (
	"path/filepath"
	"sync"

	"github.com/rotisserie/eris"
	"golang.org/x/sync/singleflight"
)

// statsCache caches parsed fileStats per file path.
// Stores both successful results and analysis errors to avoid re-parsing bad files.
type statsCache struct {
	mu    sync.Mutex
	group singleflight.Group
	stats map[string]*fileStats
	errs  map[string]error
}

var globalCache = &statsCache{
	stats: make(map[string]*fileStats),
	errs:  make(map[string]error),
}

var globalModuleCache = newModuleCache()

// getOrAnalyze returns the cached fileStats for path, parsing if necessary.
// Concurrent requests for the same path are deduplicated via singleflight.
// Both successful results and errors are cached to avoid repeated work.
func getOrAnalyze(path string) (*fileStats, error) {
	globalCache.mu.Lock()
	if s, ok := globalCache.stats[path]; ok {
		globalCache.mu.Unlock()

		return s, nil
	}

	if err, ok := globalCache.errs[path]; ok {
		globalCache.mu.Unlock()

		return nil, err
	}
	globalCache.mu.Unlock()

	result, err, _ := globalCache.group.Do(path, func() (any, error) {
		dir := filepath.Dir(path)
		modulePath := globalModuleCache.findModulePath(dir)

		s, analyzeErr := analyzeFile(path, modulePath)
		if analyzeErr != nil {
			globalCache.mu.Lock()
			globalCache.errs[path] = analyzeErr
			globalCache.mu.Unlock()

			return nil, analyzeErr
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

// ResetCacheForTesting clears the global caches. Test use only.
func ResetCacheForTesting() {
	globalCache = &statsCache{
		stats: make(map[string]*fileStats),
		errs:  make(map[string]error),
	}
	globalModuleCache = newModuleCache()

	ResetDeclCacheForTesting()
}
