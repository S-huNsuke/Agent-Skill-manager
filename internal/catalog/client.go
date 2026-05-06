package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrCatalogUnavailable  = errors.New("catalog unavailable")
	ErrIncompatibleCatalog = errors.New("catalog incompatible")
)

type Fetcher interface {
	Fetch(context.Context, string) ([]byte, error)
}

type FetchFunc func(context.Context, string) ([]byte, error)

func (f FetchFunc) Fetch(ctx context.Context, url string) ([]byte, error) {
	return f(ctx, url)
}

type Client struct {
	ClientVersion string
	Fetcher       Fetcher
	Cache         Cache
	Now           func() time.Time
}

func (c Client) Sync(ctx context.Context, url string) (SyncResult, error) {
	now := time.Now().UTC()
	if c.Now != nil {
		now = c.Now()
	}

	payload, err := c.fetch(ctx, url)
	if err != nil {
		return c.syncFromCache(now, fmt.Errorf("%w: %v", ErrCatalogUnavailable, err))
	}

	catalog, err := parseCatalog(payload)
	if err != nil {
		return c.syncFromCache(now, fmt.Errorf("%w: %v", ErrCatalogUnavailable, err))
	}

	compatibility := CheckCompatibility(catalog, c.ClientVersion)
	if !compatibility.Compatible {
		return SyncResult{
			Catalog:       catalog,
			Compatibility: compatibility,
		}, fmt.Errorf("%w: %s", ErrIncompatibleCatalog, compatibility.Reason)
	}

	cacheEntry := CacheEntry{
		Catalog:        catalog,
		FetchedAt:      now,
		CacheExpiresAt: defaultExpiry(now, catalog.CacheExpiresAt),
	}
	if c.Cache != nil {
		c.Cache.Save(cacheEntry)
	}

	return SyncResult{
		Catalog:       catalog,
		Compatibility: compatibility,
	}, nil
}

func (c Client) fetch(ctx context.Context, url string) ([]byte, error) {
	if c.Fetcher == nil {
		return nil, errors.New("fetcher is not configured")
	}
	return c.Fetcher.Fetch(ctx, url)
}

func (c Client) syncFromCache(now time.Time, cause error) (SyncResult, error) {
	if c.Cache != nil {
		entry, ok := c.Cache.Load()
		if ok && entry.ValidAt(now) {
			compatibility := CheckCompatibility(entry.Catalog, c.ClientVersion)
			if !compatibility.Compatible {
				return SyncResult{}, cause
			}
			return SyncResult{
				Catalog:           entry.Catalog,
				Compatibility:     compatibility,
				UsedCacheFallback: true,
			}, nil
		}
	}

	return SyncResult{}, cause
}

func parseCatalog(payload []byte) (Catalog, error) {
	var catalog Catalog
	if err := json.Unmarshal(payload, &catalog); err != nil {
		return Catalog{}, err
	}
	return catalog, nil
}

func defaultExpiry(now time.Time, catalogExpiry *time.Time) time.Time {
	if catalogExpiry != nil {
		return catalogExpiry.UTC()
	}
	return now.Add(24 * time.Hour)
}
