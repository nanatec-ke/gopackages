package definite

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Cache stores reference data — the product catalogue and the makes & models
// list — as raw API response bodies. Implementations must be safe for
// concurrent use. Caching is best-effort: a Set that fails is simply a miss
// on the next Get.
//
// Set Config.Cache to plug in a shared backend such as Redis, so every replica
// of a service shares one copy.
type Cache interface {
	// Get returns a live entry. The returned slice must not be modified.
	Get(key string) ([]byte, bool)
	Set(key string, value []byte, ttl time.Duration)
	Remove(key string)
	Clear()
}

/* -------------------------------------------------------------------------- */
/*  Memory                                                                    */
/* -------------------------------------------------------------------------- */

// MemoryCache is an in-process Cache. Expired entries are evicted lazily, on
// read and on write, so it starts no background goroutine.
type MemoryCache struct {
	mu    sync.Mutex
	items map[string]memoryEntry
}

type memoryEntry struct {
	value  []byte
	expiry time.Time
}

// NewMemoryCache returns an empty MemoryCache.
func NewMemoryCache() *MemoryCache { return &MemoryCache{items: make(map[string]memoryEntry)} }

// Get implements Cache.
func (c *MemoryCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(e.expiry) {
		delete(c.items, key)
		return nil, false
	}
	return e.value, true
}

// Set implements Cache. The value is copied, so the caller may reuse its slice.
func (c *MemoryCache) Set(key string, value []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, e := range c.items {
		if now.After(e.expiry) {
			delete(c.items, k)
		}
	}
	c.items[key] = memoryEntry{value: append([]byte(nil), value...), expiry: now.Add(ttl)}
}

// Remove implements Cache.
func (c *MemoryCache) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear implements Cache.
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]memoryEntry)
}

// Len returns the number of live entries.
func (c *MemoryCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	n := 0
	for _, e := range c.items {
		if !now.After(e.expiry) {
			n++
		}
	}
	return n
}

/* -------------------------------------------------------------------------- */
/*  File                                                                      */
/* -------------------------------------------------------------------------- */

// FileCache persists entries to a JSON file, so they survive restarts.
//
// The file is <dir>/definite-cache/<sha256>.json, where the digest covers the
// namespace and credentials, so each credential set and environment gets its
// own file. No credential appears in the path — unlike the TypeScript SDK,
// which base64url-encodes the password into the filename, a reversible
// encoding readable by anyone who can list the directory.
//
// The file is written with mode 0600 and replaced atomically, so a crash or a
// concurrent writer never leaves it torn. A corrupt file reads as empty.
type FileCache struct {
	mu   sync.Mutex
	path string
}

type fileEntry struct {
	Value  string `json:"value"`
	Expiry int64  `json:"expiry"` // Unix milliseconds
}

// NewFileCache returns a FileCache rooted at dir (os.TempDir() when empty).
// namespace keeps environments apart — the client passes its base URL.
func NewFileCache(dir, namespace string, creds Credentials) (*FileCache, error) {
	if dir == "" {
		dir = os.TempDir()
	}
	root := filepath.Join(dir, "definite-cache")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(namespace + "\x00" + creds.Username + "\x00" + creds.Password))
	return &FileCache{path: filepath.Join(root, hex.EncodeToString(sum[:])+".json")}, nil
}

// Path returns the backing file's location.
func (c *FileCache) Path() string { return c.path }

// Get implements Cache.
func (c *FileCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	store := c.read()
	e, ok := store[key]
	if !ok {
		return nil, false
	}
	if time.Now().UnixMilli() > e.Expiry {
		delete(store, key)
		c.write(store)
		return nil, false
	}
	return []byte(e.Value), true
}

// Set implements Cache.
func (c *FileCache) Set(key string, value []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	store := c.read()
	now := time.Now().UnixMilli()
	for k, e := range store {
		if now > e.Expiry {
			delete(store, k)
		}
	}
	store[key] = fileEntry{Value: string(value), Expiry: now + ttl.Milliseconds()}
	c.write(store)
}

// Remove implements Cache.
func (c *FileCache) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	store := c.read()
	if _, ok := store[key]; ok {
		delete(store, key)
		c.write(store)
	}
}

// Clear implements Cache.
func (c *FileCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.write(map[string]fileEntry{})
}

func (c *FileCache) read() map[string]fileEntry {
	store := map[string]fileEntry{}
	if b, err := os.ReadFile(c.path); err == nil {
		if json.Unmarshal(b, &store) != nil {
			return map[string]fileEntry{}
		}
	}
	return store
}

// write replaces the file atomically. Failures are ignored: the cache is best-effort.
func (c *FileCache) write(store map[string]fileEntry) {
	b, err := json.Marshal(store)
	if err != nil {
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(c.path), ".definite-*.tmp") // created 0600
	if err != nil {
		return
	}
	_, werr := tmp.Write(b)
	cerr := tmp.Close()
	if werr != nil || cerr != nil || os.Rename(tmp.Name(), c.path) != nil {
		_ = os.Remove(tmp.Name())
	}
}

/* -------------------------------------------------------------------------- */
/*  None                                                                      */
/* -------------------------------------------------------------------------- */

// noopCache stores nothing; selected by CacheNone.
type noopCache struct{}

func (noopCache) Get(string) ([]byte, bool)         { return nil, false }
func (noopCache) Set(string, []byte, time.Duration) {}
func (noopCache) Remove(string)                     {}
func (noopCache) Clear()                            {}
