package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestCatalogCompatibilityRejectsUnsupportedSources(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		catalog           Catalog
		clientVersion     string
		wantCompatible    bool
		wantReasonCode    string
		wantReasonMessage string
	}{
		{
			name: "unsupported major schema version",
			catalog: Catalog{
				SchemaVersion:    "2.0.0",
				MinClientVersion: "1.0.0",
			},
			clientVersion:     "1.2.0",
			wantCompatible:    false,
			wantReasonCode:    ReasonUnsupportedSchemaMajor,
			wantReasonMessage: "catalog schema major version 2 is unsupported",
		},
		{
			name: "minimum client version not met",
			catalog: Catalog{
				SchemaVersion:    "1.0.0",
				MinClientVersion: "1.5.0",
			},
			clientVersion:     "1.4.9",
			wantCompatible:    false,
			wantReasonCode:    ReasonClientTooOld,
			wantReasonMessage: "client version 1.4.9 is below required minimum 1.5.0",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := CheckCompatibility(tc.catalog, tc.clientVersion)

			if result.Compatible != tc.wantCompatible {
				t.Fatalf("Compatible = %v, want %v", result.Compatible, tc.wantCompatible)
			}
			if result.ReasonCode != tc.wantReasonCode {
				t.Fatalf("ReasonCode = %q, want %q", result.ReasonCode, tc.wantReasonCode)
			}
			if result.Reason != tc.wantReasonMessage {
				t.Fatalf("Reason = %q, want %q", result.Reason, tc.wantReasonMessage)
			}
		})
	}
}

func TestClientSyncFallsBackToCachedCatalogWhenNetworkUnavailableAndCacheValid(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 2, 22, 0, 0, 0, time.UTC)
	cachedCatalog := Catalog{
		SchemaVersion:    "1.0.0",
		StoreName:        "Cached Store",
		StoreURL:         "https://cached.example.com",
		MinClientVersion: "1.0.0",
	}
	cache := NewMemoryCache()
	cache.Save(CacheEntry{
		Catalog:        cachedCatalog,
		FetchedAt:      now.Add(-30 * time.Minute),
		CacheExpiresAt: now.Add(30 * time.Minute),
	})

	client := Client{
		ClientVersion: "1.2.0",
		Fetcher: FetchFunc(func(context.Context, string) ([]byte, error) {
			return nil, errors.New("network unavailable")
		}),
		Cache: cache,
		Now:   func() time.Time { return now },
	}

	result, err := client.Sync(context.Background(), "https://store.example.com/catalog.json")
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if !result.UsedCacheFallback {
		t.Fatalf("UsedCacheFallback = false, want true")
	}
	if result.Catalog.StoreName != cachedCatalog.StoreName {
		t.Fatalf("Catalog.StoreName = %q, want %q", result.Catalog.StoreName, cachedCatalog.StoreName)
	}
}

func TestClientSyncFallsBackToCachedCatalogWhenFetchedJSONIsMalformed(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 3, 2, 0, 0, 0, time.UTC)
	cachedCatalog := Catalog{
		SchemaVersion:    "1.0.0",
		StoreName:        "Cached Store",
		StoreURL:         "https://cached.example.com",
		MinClientVersion: "1.0.0",
	}
	cache := NewMemoryCache()
	cache.Save(CacheEntry{
		Catalog:        cachedCatalog,
		FetchedAt:      now.Add(-15 * time.Minute),
		CacheExpiresAt: now.Add(45 * time.Minute),
	})

	client := Client{
		ClientVersion: "1.2.0",
		Fetcher: FetchFunc(func(context.Context, string) ([]byte, error) {
			return []byte("{not-json"), nil
		}),
		Cache: cache,
		Now:   func() time.Time { return now },
	}

	result, err := client.Sync(context.Background(), "https://store.example.com/catalog.json")
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if !result.UsedCacheFallback {
		t.Fatalf("UsedCacheFallback = false, want true")
	}
	if result.Catalog.StoreName != cachedCatalog.StoreName {
		t.Fatalf("Catalog.StoreName = %q, want %q", result.Catalog.StoreName, cachedCatalog.StoreName)
	}
}

func TestClientSyncBlocksIncompatibleFetchedCatalogEvenWhenCacheExists(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 3, 1, 0, 0, 0, time.UTC)
	cache := NewMemoryCache()
	cache.Save(CacheEntry{
		Catalog: Catalog{
			SchemaVersion:    "1.0.0",
			StoreName:        "Cached Store",
			MinClientVersion: "1.0.0",
		},
		FetchedAt:      now.Add(-10 * time.Minute),
		CacheExpiresAt: now.Add(1 * time.Hour),
	})

	payload, err := json.Marshal(Catalog{
		SchemaVersion:    "2.0.0",
		StoreName:        "Fetched Store",
		MinClientVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	client := Client{
		ClientVersion: "1.2.0",
		Fetcher: FetchFunc(func(context.Context, string) ([]byte, error) {
			return payload, nil
		}),
		Cache: cache,
		Now:   func() time.Time { return now },
	}

	_, syncErr := client.Sync(context.Background(), "https://store.example.com/catalog.json")
	if !errors.Is(syncErr, ErrIncompatibleCatalog) {
		t.Fatalf("Sync() error = %v, want ErrIncompatibleCatalog", syncErr)
	}
}

func TestClientSyncDoesNotUseExpiredCacheFallback(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 3, 1, 0, 0, 0, time.UTC)
	cache := NewMemoryCache()
	cache.Save(CacheEntry{
		Catalog: Catalog{
			SchemaVersion:    "1.0.0",
			StoreName:        "Expired Cache",
			MinClientVersion: "1.0.0",
		},
		FetchedAt:      now.Add(-2 * time.Hour),
		CacheExpiresAt: now.Add(-1 * time.Minute),
	})

	client := Client{
		ClientVersion: "1.2.0",
		Fetcher: FetchFunc(func(context.Context, string) ([]byte, error) {
			return nil, errors.New("network unavailable")
		}),
		Cache: cache,
		Now:   func() time.Time { return now },
	}

	_, err := client.Sync(context.Background(), "https://store.example.com/catalog.json")
	if !errors.Is(err, ErrCatalogUnavailable) {
		t.Fatalf("Sync() error = %v, want ErrCatalogUnavailable", err)
	}
}

func TestClientSyncDoesNotUseIncompatibleCacheFallback(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 3, 1, 0, 0, 0, time.UTC)
	cache := NewMemoryCache()
	cache.Save(CacheEntry{
		Catalog: Catalog{
			SchemaVersion:    "1.0.0",
			StoreName:        "Incompatible Cache",
			MinClientVersion: "9.0.0",
		},
		FetchedAt:      now.Add(-10 * time.Minute),
		CacheExpiresAt: now.Add(1 * time.Hour),
	})

	client := Client{
		ClientVersion: "1.2.0",
		Fetcher: FetchFunc(func(context.Context, string) ([]byte, error) {
			return nil, errors.New("network unavailable")
		}),
		Cache: cache,
		Now:   func() time.Time { return now },
	}

	_, err := client.Sync(context.Background(), "https://store.example.com/catalog.json")
	if !errors.Is(err, ErrCatalogUnavailable) {
		t.Fatalf("Sync() error = %v, want ErrCatalogUnavailable", err)
	}
}

func TestClientSyncBlocksWhenOfflineWithoutValidCache(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 2, 22, 0, 0, 0, time.UTC)
	client := Client{
		ClientVersion: "1.2.0",
		Fetcher: FetchFunc(func(context.Context, string) ([]byte, error) {
			return nil, errors.New("network unavailable")
		}),
		Cache: NewMemoryCache(),
		Now:   func() time.Time { return now },
	}

	_, err := client.Sync(context.Background(), "https://store.example.com/catalog.json")
	if !errors.Is(err, ErrCatalogUnavailable) {
		t.Fatalf("Sync() error = %v, want ErrCatalogUnavailable", err)
	}
}
