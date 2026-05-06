package installer

import "context"

type Validator interface {
	Validate(context.Context, ExtractedPackage, PackageRef) error
}

type ValidateFunc func(context.Context, ExtractedPackage, PackageRef) error

func (f ValidateFunc) Validate(ctx context.Context, extracted ExtractedPackage, pkg PackageRef) error {
	return f(ctx, extracted, pkg)
}
