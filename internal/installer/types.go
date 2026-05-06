package installer

type PackageRef struct {
	Version        string
	ChecksumSHA256 string
	DownloadBytes  int64
	ExtractBytes   int64
}

type InstalledState struct {
	Version        string
	ChecksumSHA256 string
	IsValid        bool
}

type PreflightRequest struct {
	NetworkAvailable   bool
	HasCachedPackage   bool
	CachedPackageValid bool
	CachedPackagePath  string
	DownloadSizeBytes  int64
	ExtractSizeBytes   int64
	InstalledVersion   string
	RequestedVersion   string
	InstalledChecksum  string
	RequestedChecksum  string
	ExistingInstallOK  bool
}

type PreflightResult struct {
	EnoughDiskSpace     bool
	NeedsFullReinstall  bool
	CanUseCachedPackage bool
	FailureReason       string
}

type InstallRequest struct {
	Package            PackageRef
	Current            InstalledState
	NetworkAvailable   bool
	HasCachedPackage   bool
	CachedPackageValid bool
	CachedPackagePath  string
}

type InstallResult struct {
	Preflight PreflightResult
	Strategy  string
}

const (
	StrategyIncremental   = "incremental"
	StrategyFullReinstall = "full_reinstall"
)
