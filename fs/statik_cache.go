package fs

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// statikCache is a struct that represents a cache of statik.json files.
type statikCache struct {
	baseUrl   string
	cache     map[string]statikCacheEl
	cacheLock sync.RWMutex
}

func newStatikCache(baseUrl string) *statikCache {
	return &statikCache{
		baseUrl: baseUrl,
		cache:   make(map[string]statikCacheEl),
	}
}

// statikCacheEl represents a cached statik.json file and its expiration time.
type statikCacheEl struct {
	statik Statik
	exp    time.Time
}

// Get returns the Statik struct for the statik.json file in the directory
// specified by path.
//
// If the statik.json file is not cached, it is fetched from the remote server,
// cached and returned.
//
// The function is safe for concurrent use, as it uses a RW mutex to protect the
// cache.
func (m *statikCache) Get(path string) (Statik, error) {
	// check cache
	m.cacheLock.RLock()
	cache, contentOk := m.cache[path]
	m.cacheLock.RUnlock()

	if contentOk && cache.exp.After(time.Now()) {
		// cache hit
		return cache.statik, nil
	}

	if contentOk {
		// cache expired
		m.cacheLock.Lock()
		delete(m.cache, path)
		m.cacheLock.Unlock()
	}

	// cache miss
	slog.Debug("caching statik.json for path", "path", path)

	response, err := http.Get(m.baseUrl + path + "/statik.json")
	if err != nil {
		return Statik{}, fmt.Errorf("error getting statik.json: %w", err)
	}

	statik := Statik{}
	err = json.NewDecoder(response.Body).Decode(&statik)
	if err != nil {
		return Statik{}, fmt.Errorf("error decoding statik.json: %w", err)
	}

	err = response.Body.Close()
	if err != nil {
		return Statik{}, fmt.Errorf("error closing response body: %w", err)
	}

	// populate cache
	m.cacheLock.Lock()
	m.cache[path] = statikCacheEl{statik, time.Now().Add(StatikCachingTime)}
	m.cacheLock.Unlock()

	return statik, nil
}
