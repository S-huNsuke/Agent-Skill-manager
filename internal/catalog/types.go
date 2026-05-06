package catalog

import "time"

const supportedSchemaMajor = 1

type Catalog struct {
	SchemaVersion    string         `json:"schema_version"`
	StoreName        string         `json:"store_name"`
	StoreURL         string         `json:"store_url"`
	MinClientVersion string         `json:"min_client_version"`
	CacheExpiresAt   *time.Time     `json:"cache_expires_at,omitempty"`
	Skills           []CatalogSkill `json:"skills"`
}

type CatalogSkill struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	Author          string   `json:"author"`
	Description     string   `json:"description"`
	Homepage        string   `json:"homepage"`
	PackageURL      string   `json:"package_url"`
	ChecksumSHA256  string   `json:"checksum_sha256"`
	SupportedAgents []string `json:"supported_agents"`
	SchemaVersion   string   `json:"schema_version"`
}

type SyncResult struct {
	Catalog           Catalog
	Compatibility     CompatibilityResult
	UsedCacheFallback bool
}
