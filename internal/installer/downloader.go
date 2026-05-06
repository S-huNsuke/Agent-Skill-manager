package installer

import "context"

type DownloadedPackage struct {
	Path string
}

type Downloader interface {
	Download(context.Context, PackageRef) (DownloadedPackage, error)
}

type DownloadFunc func(context.Context, PackageRef) (DownloadedPackage, error)

func (f DownloadFunc) Download(ctx context.Context, pkg PackageRef) (DownloadedPackage, error) {
	return f(ctx, pkg)
}
