package installer

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestServicePreflightBlocksWhenOfflineWithoutCache(t *testing.T) {
	t.Parallel()

	service := Service{
		Disk: StaticDiskEstimator{
			BytesAvailable: 1024,
		},
	}

	result, err := service.Preflight(context.Background(), PreflightRequest{
		NetworkAvailable:   false,
		HasCachedPackage:   false,
		DownloadSizeBytes:  128,
		ExtractSizeBytes:   128,
		InstalledVersion:   "1.0.0",
		RequestedVersion:   "1.0.0",
		InstalledChecksum:  "abc",
		RequestedChecksum:  "abc",
		ExistingInstallOK:  true,
		CachedPackageValid: false,
	})
	if err != nil {
		t.Fatalf("Preflight() error = %v", err)
	}
	if !result.EnoughDiskSpace {
		t.Fatalf("EnoughDiskSpace = false, want true")
	}
	if result.FailureReason != "network unavailable and no valid cached package available" {
		t.Fatalf("FailureReason = %q", result.FailureReason)
	}
}

func TestServicePreflightStillRequiresDiskEstimatorWhenOfflineWithoutCache(t *testing.T) {
	t.Parallel()

	service := Service{}

	result, err := service.Preflight(context.Background(), PreflightRequest{
		NetworkAvailable:   false,
		HasCachedPackage:   false,
		CachedPackageValid: false,
		DownloadSizeBytes:  128,
		ExtractSizeBytes:   128,
		InstalledVersion:   "1.0.0",
		RequestedVersion:   "1.0.0",
		InstalledChecksum:  "abc",
		RequestedChecksum:  "abc",
		ExistingInstallOK:  true,
	})
	if !errors.Is(err, ErrDiskEstimatorRequired) {
		t.Fatalf("Preflight() error = %v, want ErrDiskEstimatorRequired", err)
	}
	if result.FailureReason != "" {
		t.Fatalf("FailureReason = %q, want empty", result.FailureReason)
	}
}

func TestServicePreflightFailsExplicitlyWhenDiskEstimatorIsMissing(t *testing.T) {
	t.Parallel()

	service := Service{}

	result, err := service.Preflight(context.Background(), PreflightRequest{
		NetworkAvailable:   true,
		HasCachedPackage:   false,
		CachedPackageValid: false,
		DownloadSizeBytes:  128,
		ExtractSizeBytes:   128,
		InstalledVersion:   "1.0.0",
		RequestedVersion:   "1.0.0",
		InstalledChecksum:  "abc",
		RequestedChecksum:  "abc",
		ExistingInstallOK:  true,
	})
	if !errors.Is(err, ErrDiskEstimatorRequired) {
		t.Fatalf("Preflight() error = %v, want ErrDiskEstimatorRequired", err)
	}
	if result.FailureReason != "" {
		t.Fatalf("FailureReason = %q, want empty", result.FailureReason)
	}
}

func TestServicePreflightTreatsInvalidCachedPackageAsUnavailable(t *testing.T) {
	t.Parallel()

	service := Service{
		Disk: StaticDiskEstimator{
			BytesAvailable: 1024,
		},
	}

	result, err := service.Preflight(context.Background(), PreflightRequest{
		NetworkAvailable:   false,
		HasCachedPackage:   true,
		CachedPackageValid: false,
		DownloadSizeBytes:  128,
		ExtractSizeBytes:   128,
		InstalledVersion:   "1.0.0",
		RequestedVersion:   "1.0.0",
		InstalledChecksum:  "abc",
		RequestedChecksum:  "abc",
		ExistingInstallOK:  true,
	})
	if err != nil {
		t.Fatalf("Preflight() error = %v", err)
	}
	if result.CanUseCachedPackage {
		t.Fatalf("CanUseCachedPackage = true, want false")
	}
	if result.FailureReason != "network unavailable and no valid cached package available" {
		t.Fatalf("FailureReason = %q", result.FailureReason)
	}
}

func TestServicePreflightAllowsSafeIncrementalUpdate(t *testing.T) {
	t.Parallel()

	service := Service{
		Disk: StaticDiskEstimator{
			BytesAvailable: 4096,
		},
	}

	result, err := service.Preflight(context.Background(), PreflightRequest{
		NetworkAvailable:   true,
		HasCachedPackage:   false,
		CachedPackageValid: false,
		DownloadSizeBytes:  512,
		ExtractSizeBytes:   512,
		InstalledVersion:   "1.0.0",
		RequestedVersion:   "1.0.0",
		InstalledChecksum:  "abc",
		RequestedChecksum:  "abc",
		ExistingInstallOK:  true,
	})
	if err != nil {
		t.Fatalf("Preflight() error = %v", err)
	}
	if !result.EnoughDiskSpace {
		t.Fatalf("EnoughDiskSpace = false, want true")
	}
	if result.NeedsFullReinstall {
		t.Fatalf("NeedsFullReinstall = true, want false")
	}
}

func TestServicePreflightRequiresFullReinstallWhenInstalledPackageIsUnsafe(t *testing.T) {
	t.Parallel()

	service := Service{
		Disk: StaticDiskEstimator{
			BytesAvailable: 4096,
		},
	}

	result, err := service.Preflight(context.Background(), PreflightRequest{
		NetworkAvailable:   true,
		HasCachedPackage:   true,
		DownloadSizeBytes:  512,
		ExtractSizeBytes:   512,
		InstalledVersion:   "1.0.0",
		RequestedVersion:   "1.1.0",
		InstalledChecksum:  "abc",
		RequestedChecksum:  "def",
		ExistingInstallOK:  false,
		CachedPackageValid: true,
		CachedPackagePath:  "/tmp/cache/package.tgz",
	})
	if err != nil {
		t.Fatalf("Preflight() error = %v", err)
	}
	if !result.EnoughDiskSpace {
		t.Fatalf("EnoughDiskSpace = false, want true")
	}
	if !result.NeedsFullReinstall {
		t.Fatalf("NeedsFullReinstall = false, want true")
	}
	if !result.CanUseCachedPackage {
		t.Fatalf("CanUseCachedPackage = false, want true")
	}
}

func TestServiceInstallChoosesIncrementalWhenSafe(t *testing.T) {
	t.Parallel()

	service := Service{
		Disk: StaticDiskEstimator{
			BytesAvailable: 4096,
		},
		Downloader: DownloadFunc(func(context.Context, PackageRef) (DownloadedPackage, error) {
			return DownloadedPackage{Path: "/tmp/network-package.tgz"}, nil
		}),
		Extractor: ExtractFunc(func(context.Context, DownloadedPackage) (ExtractedPackage, error) {
			return ExtractedPackage{Path: "/tmp/extracted"}, nil
		}),
		Validator: ValidateFunc(func(context.Context, ExtractedPackage, PackageRef) error {
			return nil
		}),
	}

	result, err := service.Install(context.Background(), InstallRequest{
		Package: PackageRef{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			DownloadBytes:  256,
			ExtractBytes:   256,
		},
		Current: InstalledState{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			IsValid:        true,
		},
		NetworkAvailable:   true,
		HasCachedPackage:   false,
		CachedPackageValid: false,
	})
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if result.Strategy != StrategyIncremental {
		t.Fatalf("Strategy = %q, want %q", result.Strategy, StrategyIncremental)
	}
}

func TestServiceInstallChoosesFullReinstallWhenVersionChanges(t *testing.T) {
	t.Parallel()

	service := Service{
		Disk: StaticDiskEstimator{
			BytesAvailable: 4096,
		},
		Downloader: DownloadFunc(func(context.Context, PackageRef) (DownloadedPackage, error) {
			return DownloadedPackage{Path: "/tmp/network-package.tgz"}, nil
		}),
		Extractor: ExtractFunc(func(context.Context, DownloadedPackage) (ExtractedPackage, error) {
			return ExtractedPackage{Path: "/tmp/extracted"}, nil
		}),
		Validator: ValidateFunc(func(context.Context, ExtractedPackage, PackageRef) error {
			return nil
		}),
	}

	result, err := service.Install(context.Background(), InstallRequest{
		Package: PackageRef{
			Version:        "1.1.0",
			ChecksumSHA256: "def",
			DownloadBytes:  256,
			ExtractBytes:   256,
		},
		Current: InstalledState{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			IsValid:        true,
		},
		NetworkAvailable:   true,
		HasCachedPackage:   false,
		CachedPackageValid: false,
	})
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if result.Strategy != StrategyFullReinstall {
		t.Fatalf("Strategy = %q, want %q", result.Strategy, StrategyFullReinstall)
	}
}

func TestServiceInstallReturnsPreflightFailure(t *testing.T) {
	t.Parallel()

	service := Service{
		Disk: StaticDiskEstimator{
			BytesAvailable: 128,
		},
		Downloader: DownloadFunc(func(context.Context, PackageRef) (DownloadedPackage, error) {
			return DownloadedPackage{}, errors.New("should not download")
		}),
	}

	_, err := service.Install(context.Background(), InstallRequest{
		Package: PackageRef{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			DownloadBytes:  128,
			ExtractBytes:   128,
		},
		Current: InstalledState{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			IsValid:        true,
		},
		NetworkAvailable:   true,
		HasCachedPackage:   false,
		CachedPackageValid: false,
	})
	if !errors.Is(err, ErrPreflightFailed) {
		t.Fatalf("Install() error = %v, want ErrPreflightFailed", err)
	}
}

func TestServiceInstallFailsExplicitlyWhenDiskEstimatorIsMissing(t *testing.T) {
	t.Parallel()

	service := Service{
		Downloader: DownloadFunc(func(context.Context, PackageRef) (DownloadedPackage, error) {
			return DownloadedPackage{}, errors.New("should not download")
		}),
	}

	result, err := service.Install(context.Background(), InstallRequest{
		Package: PackageRef{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			DownloadBytes:  128,
			ExtractBytes:   128,
		},
		Current: InstalledState{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			IsValid:        true,
		},
		NetworkAvailable:   true,
		HasCachedPackage:   false,
		CachedPackageValid: false,
	})
	if !errors.Is(err, ErrDiskEstimatorRequired) {
		t.Fatalf("Install() error = %v, want ErrDiskEstimatorRequired", err)
	}
	if result.Preflight.EnoughDiskSpace {
		t.Fatalf("EnoughDiskSpace = true, want false")
	}
}

func TestServiceInstallFailsWhenDownloadRequiredButDownloaderMissing(t *testing.T) {
	t.Parallel()

	service := Service{
		Disk: StaticDiskEstimator{
			BytesAvailable: 4096,
		},
		Extractor: ExtractFunc(func(context.Context, DownloadedPackage) (ExtractedPackage, error) {
			return ExtractedPackage{}, errors.New("should not extract")
		}),
		Validator: ValidateFunc(func(context.Context, ExtractedPackage, PackageRef) error {
			return errors.New("should not validate")
		}),
	}

	_, err := service.Install(context.Background(), InstallRequest{
		Package: PackageRef{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			DownloadBytes:  256,
			ExtractBytes:   256,
		},
		Current: InstalledState{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			IsValid:        true,
		},
		NetworkAvailable:   true,
		HasCachedPackage:   false,
		CachedPackageValid: false,
	})
	if !errors.Is(err, ErrDownloaderRequired) {
		t.Fatalf("Install() error = %v, want ErrDownloaderRequired", err)
	}
}

func TestServiceInstallFailsBeforeDownloadWhenExtractorMissing(t *testing.T) {
	t.Parallel()

	downloadCalled := false
	service := Service{
		Disk: StaticDiskEstimator{
			BytesAvailable: 4096,
		},
		Downloader: DownloadFunc(func(context.Context, PackageRef) (DownloadedPackage, error) {
			downloadCalled = true
			return DownloadedPackage{Path: "/tmp/network-package.tgz"}, nil
		}),
		Validator: ValidateFunc(func(context.Context, ExtractedPackage, PackageRef) error {
			return errors.New("should not validate")
		}),
	}

	_, err := service.Install(context.Background(), InstallRequest{
		Package: PackageRef{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			DownloadBytes:  256,
			ExtractBytes:   256,
		},
		Current: InstalledState{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			IsValid:        true,
		},
		NetworkAvailable:   true,
		HasCachedPackage:   false,
		CachedPackageValid: false,
	})
	if !errors.Is(err, ErrExtractorRequired) {
		t.Fatalf("Install() error = %v, want ErrExtractorRequired", err)
	}
	if downloadCalled {
		t.Fatalf("downloadCalled = true, want false")
	}
}

func TestServiceInstallFailsBeforeDownloadWhenValidatorMissing(t *testing.T) {
	t.Parallel()

	downloadCalled := false
	extractCalled := false
	service := Service{
		Disk: StaticDiskEstimator{
			BytesAvailable: 4096,
		},
		Downloader: DownloadFunc(func(context.Context, PackageRef) (DownloadedPackage, error) {
			downloadCalled = true
			return DownloadedPackage{Path: "/tmp/network-package.tgz"}, nil
		}),
		Extractor: ExtractFunc(func(context.Context, DownloadedPackage) (ExtractedPackage, error) {
			extractCalled = true
			return ExtractedPackage{Path: "/tmp/extracted"}, nil
		}),
	}

	_, err := service.Install(context.Background(), InstallRequest{
		Package: PackageRef{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			DownloadBytes:  256,
			ExtractBytes:   256,
		},
		Current: InstalledState{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			IsValid:        true,
		},
		NetworkAvailable:   true,
		HasCachedPackage:   false,
		CachedPackageValid: false,
	})
	if !errors.Is(err, ErrValidatorRequired) {
		t.Fatalf("Install() error = %v, want ErrValidatorRequired", err)
	}
	if downloadCalled {
		t.Fatalf("downloadCalled = true, want false")
	}
	if extractCalled {
		t.Fatalf("extractCalled = true, want false")
	}
}

func TestServiceInstallTreatsEmptyCachedPackagePathAsUnavailableDuringPreflight(t *testing.T) {
	t.Parallel()

	service := Service{
		Disk: StaticDiskEstimator{
			BytesAvailable: 4096,
		},
		Extractor: ExtractFunc(func(context.Context, DownloadedPackage) (ExtractedPackage, error) {
			return ExtractedPackage{}, errors.New("should not extract")
		}),
		Validator: ValidateFunc(func(context.Context, ExtractedPackage, PackageRef) error {
			return errors.New("should not validate")
		}),
	}

	result, err := service.Install(context.Background(), InstallRequest{
		Package: PackageRef{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			DownloadBytes:  256,
			ExtractBytes:   256,
		},
		Current: InstalledState{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			IsValid:        true,
		},
		NetworkAvailable:   false,
		HasCachedPackage:   true,
		CachedPackageValid: true,
		CachedPackagePath:  "",
	})
	if !errors.Is(err, ErrPreflightFailed) {
		t.Fatalf("Install() error = %v, want ErrPreflightFailed", err)
	}
	if result.Preflight.CanUseCachedPackage {
		t.Fatalf("CanUseCachedPackage = true, want false")
	}
	if !result.Preflight.EnoughDiskSpace {
		t.Fatalf("EnoughDiskSpace = false, want true")
	}
	if result.Preflight.FailureReason != "network unavailable and no valid cached package available" {
		t.Fatalf("FailureReason = %q", result.Preflight.FailureReason)
	}
}

func TestServiceInstallPreservesContextWhenExtractorFails(t *testing.T) {
	t.Parallel()

	service := Service{
		Disk: StaticDiskEstimator{
			BytesAvailable: 4096,
		},
		Downloader: DownloadFunc(func(context.Context, PackageRef) (DownloadedPackage, error) {
			return DownloadedPackage{Path: "/tmp/network-package.tgz"}, nil
		}),
		Extractor: ExtractFunc(func(context.Context, DownloadedPackage) (ExtractedPackage, error) {
			return ExtractedPackage{}, errors.New("extract boom")
		}),
		Validator: ValidateFunc(func(context.Context, ExtractedPackage, PackageRef) error {
			return errors.New("should not validate")
		}),
	}

	result, err := service.Install(context.Background(), InstallRequest{
		Package: PackageRef{
			Version:        "1.1.0",
			ChecksumSHA256: "def",
			DownloadBytes:  256,
			ExtractBytes:   256,
		},
		Current: InstalledState{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			IsValid:        true,
		},
		NetworkAvailable:   true,
		HasCachedPackage:   false,
		CachedPackageValid: false,
	})
	if err == nil || err.Error() != "extract boom" {
		t.Fatalf("Install() error = %v, want extract boom", err)
	}
	if !result.Preflight.EnoughDiskSpace {
		t.Fatalf("EnoughDiskSpace = false, want true")
	}
	if result.Strategy != StrategyFullReinstall {
		t.Fatalf("Strategy = %q, want %q", result.Strategy, StrategyFullReinstall)
	}
}

func TestServiceInstallPreservesContextWhenValidatorFails(t *testing.T) {
	t.Parallel()

	service := Service{
		Disk: StaticDiskEstimator{
			BytesAvailable: 4096,
		},
		Downloader: DownloadFunc(func(context.Context, PackageRef) (DownloadedPackage, error) {
			return DownloadedPackage{Path: "/tmp/network-package.tgz"}, nil
		}),
		Extractor: ExtractFunc(func(context.Context, DownloadedPackage) (ExtractedPackage, error) {
			return ExtractedPackage{Path: "/tmp/extracted"}, nil
		}),
		Validator: ValidateFunc(func(context.Context, ExtractedPackage, PackageRef) error {
			return errors.New("validate boom")
		}),
	}

	result, err := service.Install(context.Background(), InstallRequest{
		Package: PackageRef{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			DownloadBytes:  256,
			ExtractBytes:   256,
		},
		Current: InstalledState{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			IsValid:        true,
		},
		NetworkAvailable:   true,
		HasCachedPackage:   false,
		CachedPackageValid: false,
	})
	if err == nil || err.Error() != "validate boom" {
		t.Fatalf("Install() error = %v, want validate boom", err)
	}
	if !result.Preflight.EnoughDiskSpace {
		t.Fatalf("EnoughDiskSpace = false, want true")
	}
	if result.Strategy != StrategyIncremental {
		t.Fatalf("Strategy = %q, want %q", result.Strategy, StrategyIncremental)
	}
}

func TestServiceInstallRunsExtractorAndValidatorForCachedPackage(t *testing.T) {
	t.Parallel()

	var calls []string
	service := Service{
		Disk: StaticDiskEstimator{
			BytesAvailable: 4096,
		},
		Extractor: ExtractFunc(func(_ context.Context, pkg DownloadedPackage) (ExtractedPackage, error) {
			calls = append(calls, "extract:"+pkg.Path)
			return ExtractedPackage{Path: "/tmp/cached-extracted"}, nil
		}),
		Validator: ValidateFunc(func(_ context.Context, pkg ExtractedPackage, ref PackageRef) error {
			calls = append(calls, "validate:"+pkg.Path+":"+ref.Version)
			return nil
		}),
	}

	result, err := service.Install(context.Background(), InstallRequest{
		Package: PackageRef{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			DownloadBytes:  256,
			ExtractBytes:   256,
		},
		Current: InstalledState{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			IsValid:        true,
		},
		NetworkAvailable:   false,
		HasCachedPackage:   true,
		CachedPackageValid: true,
		CachedPackagePath:  "/tmp/cache/package.tgz",
	})
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if result.Strategy != StrategyIncremental {
		t.Fatalf("Strategy = %q, want %q", result.Strategy, StrategyIncremental)
	}
	wantCalls := []string{
		"extract:/tmp/cache/package.tgz",
		"validate:/tmp/cached-extracted:1.0.0",
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("calls = %#v, want %#v", calls, wantCalls)
	}
}

func TestServiceInstallRunsDownloaderExtractorAndValidatorForNetworkPackage(t *testing.T) {
	t.Parallel()

	var calls []string
	service := Service{
		Disk: StaticDiskEstimator{
			BytesAvailable: 4096,
		},
		Downloader: DownloadFunc(func(_ context.Context, pkg PackageRef) (DownloadedPackage, error) {
			calls = append(calls, "download:"+pkg.Version)
			return DownloadedPackage{Path: "/tmp/network/package.tgz"}, nil
		}),
		Extractor: ExtractFunc(func(_ context.Context, pkg DownloadedPackage) (ExtractedPackage, error) {
			calls = append(calls, "extract:"+pkg.Path)
			return ExtractedPackage{Path: "/tmp/network/extracted"}, nil
		}),
		Validator: ValidateFunc(func(_ context.Context, pkg ExtractedPackage, ref PackageRef) error {
			calls = append(calls, "validate:"+pkg.Path+":"+ref.Version)
			return nil
		}),
	}

	result, err := service.Install(context.Background(), InstallRequest{
		Package: PackageRef{
			Version:        "1.1.0",
			ChecksumSHA256: "def",
			DownloadBytes:  256,
			ExtractBytes:   256,
		},
		Current: InstalledState{
			Version:        "1.0.0",
			ChecksumSHA256: "abc",
			IsValid:        true,
		},
		NetworkAvailable:   true,
		HasCachedPackage:   false,
		CachedPackageValid: false,
	})
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if result.Strategy != StrategyFullReinstall {
		t.Fatalf("Strategy = %q, want %q", result.Strategy, StrategyFullReinstall)
	}
	wantCalls := []string{
		"download:1.1.0",
		"extract:/tmp/network/package.tgz",
		"validate:/tmp/network/extracted:1.1.0",
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("calls = %#v, want %#v", calls, wantCalls)
	}
}
