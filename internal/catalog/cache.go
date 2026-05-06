package catalog

import "time"

type CacheEntry struct {
	Catalog        Catalog
	FetchedAt      time.Time
	CacheExpiresAt time.Time
}

func (e CacheEntry) ValidAt(now time.Time) bool {
	return !e.CacheExpiresAt.IsZero() && now.Before(e.CacheExpiresAt)
}

type Cache interface {
	Load() (CacheEntry, bool)
	Save(CacheEntry)
}

type MemoryCache struct {
	entry CacheEntry
	ok    bool
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{}
}

func (c *MemoryCache) Load() (CacheEntry, bool) {
	return c.entry, c.ok
}

func (c *MemoryCache) Save(entry CacheEntry) {
	c.entry = entry
	c.ok = true
}
