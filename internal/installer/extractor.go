package installer

import "context"

type ExtractedPackage struct {
	Path string
}

type Extractor interface {
	Extract(context.Context, DownloadedPackage) (ExtractedPackage, error)
}

type ExtractFunc func(context.Context, DownloadedPackage) (ExtractedPackage, error)

func (f ExtractFunc) Extract(ctx context.Context, pkg DownloadedPackage) (ExtractedPackage, error) {
	return f(ctx, pkg)
}
