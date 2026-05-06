package installer

import (
	"context"
	"errors"
	"fmt"
)

var ErrPreflightFailed = errors.New("preflight failed")
var ErrDiskEstimatorRequired = errors.New("disk estimator required")
var ErrDownloaderRequired = errors.New("downloader required")
var ErrExtractorRequired = errors.New("extractor required")
var ErrValidatorRequired = errors.New("validator required")
var ErrCachedPackagePathRequired = errors.New("cached package path required")

type DiskEstimator interface {
	AvailableBytes(context.Context) (int64, error)
}

type StaticDiskEstimator struct {
	BytesAvailable int64
}

func (s StaticDiskEstimator) AvailableBytes(context.Context) (int64, error) {
	return s.BytesAvailable, nil
}

type Service struct {
	Disk       DiskEstimator
	Downloader Downloader
	Extractor  Extractor
	Validator  Validator
}

func (s Service) Preflight(ctx context.Context, req PreflightRequest) (PreflightResult, error) {
	result := PreflightResult{
		NeedsFullReinstall:  requiresFullReinstall(req),
		CanUseCachedPackage: canUseCachedPackage(req),
	}

	available, err := s.availableBytes(ctx)
	if err != nil {
		return PreflightResult{}, err
	}

	required := req.ExtractSizeBytes
	if !result.CanUseCachedPackage {
		required += req.DownloadSizeBytes
	}

	if available < required {
		result.FailureReason = fmt.Sprintf("insufficient disk space: need %d bytes, have %d bytes", required, available)
	}

	if available >= required {
		result.EnoughDiskSpace = true
	}

	if !req.NetworkAvailable && !result.CanUseCachedPackage {
		result.FailureReason = "network unavailable and no valid cached package available"
		return result, nil
	}

	if !result.EnoughDiskSpace {
		return result, nil
	}

	return result, nil
}

func (s Service) Install(ctx context.Context, req InstallRequest) (InstallResult, error) {
	preflight, err := s.Preflight(ctx, PreflightRequest{
		NetworkAvailable:   req.NetworkAvailable,
		HasCachedPackage:   req.HasCachedPackage,
		CachedPackageValid: req.CachedPackageValid,
		CachedPackagePath:  req.CachedPackagePath,
		DownloadSizeBytes:  req.Package.DownloadBytes,
		ExtractSizeBytes:   req.Package.ExtractBytes,
		InstalledVersion:   req.Current.Version,
		RequestedVersion:   req.Package.Version,
		InstalledChecksum:  req.Current.ChecksumSHA256,
		RequestedChecksum:  req.Package.ChecksumSHA256,
		ExistingInstallOK:  req.Current.IsValid,
	})
	if err != nil {
		return InstallResult{}, err
	}
	if preflight.FailureReason != "" {
		return InstallResult{Preflight: preflight}, fmt.Errorf("%w: %s", ErrPreflightFailed, preflight.FailureReason)
	}

	result := InstallResult{Preflight: preflight}
	if preflight.NeedsFullReinstall {
		result.Strategy = StrategyFullReinstall
	} else {
		result.Strategy = StrategyIncremental
	}

	if s.Extractor == nil {
		return InstallResult{Preflight: preflight, Strategy: result.Strategy}, ErrExtractorRequired
	}
	if s.Validator == nil {
		return InstallResult{Preflight: preflight, Strategy: result.Strategy}, ErrValidatorRequired
	}

	downloaded, err := s.obtainPackage(ctx, req, preflight)
	if err != nil {
		return InstallResult{Preflight: preflight, Strategy: result.Strategy}, err
	}

	extracted, err := s.Extractor.Extract(ctx, downloaded)
	if err != nil {
		return InstallResult{Preflight: preflight, Strategy: result.Strategy}, err
	}

	if err := s.Validator.Validate(ctx, extracted, req.Package); err != nil {
		return InstallResult{Preflight: preflight, Strategy: result.Strategy}, err
	}

	return result, nil
}

func (s Service) availableBytes(ctx context.Context) (int64, error) {
	if s.Disk == nil {
		return 0, ErrDiskEstimatorRequired
	}
	return s.Disk.AvailableBytes(ctx)
}

func requiresFullReinstall(req PreflightRequest) bool {
	if !req.ExistingInstallOK {
		return true
	}
	if req.InstalledVersion != req.RequestedVersion {
		return true
	}
	return req.InstalledChecksum != req.RequestedChecksum
}

func canUseCachedPackage(req PreflightRequest) bool {
	return req.HasCachedPackage && req.CachedPackageValid && req.CachedPackagePath != ""
}

func (s Service) obtainPackage(ctx context.Context, req InstallRequest, preflight PreflightResult) (DownloadedPackage, error) {
	if preflight.CanUseCachedPackage {
		if req.CachedPackagePath == "" {
			return DownloadedPackage{}, ErrCachedPackagePathRequired
		}
		return DownloadedPackage{Path: req.CachedPackagePath}, nil
	}

	if s.Downloader == nil {
		return DownloadedPackage{}, ErrDownloaderRequired
	}

	return s.Downloader.Download(ctx, req.Package)
}
