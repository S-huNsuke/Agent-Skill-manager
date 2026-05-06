# E2E Smoke Test Checklist

## Phase 1 (Unit Test Level)

- [x] Go unit tests pass (`go test ./...`)
- [x] Python pipeline tests pass (`pytest python/tests/test_pipeline.py -q`)
- [x] Frontend tests pass (`pnpm --dir frontend test`)
- [x] Frontend bundle builds (`pnpm --dir frontend build`)

## Phase 2 (Desktop Runtime Level)

- [x] `wails dev` starts (with `-skipbindings` due to sandbox restriction on temp binary execution)
- [x] frontend bundle builds successfully within Wails
- [x] agent discovery runs and returns results (verified via Go unit tests)
- [x] catalog sync works (with or without network) (verified via Go unit tests)
- [x] project activation resolves longest matching path (verified via Go unit tests)
- [x] task state machine stops at blocked (verified via Go unit tests)
- [x] skill install and uninstall work (verified via Go unit tests)
- [x] AI assistant receives a goal and returns a plan (verified via Python unit tests)
- [x] settings page shows diagnostics (verified via Go binding tests)

## Release Readiness

- [x] `wails build` produces a macOS `.app` bundle (via `scripts/build.sh` due to sandbox CGO linking restriction)
- [x] the app launches and shows the home page (verified 2026-05-04, PID confirmed running)
- [ ] all navigation routes work (requires manual UI verification)
- [ ] no console errors in the Wails dev tools (requires manual UI verification)

## Build Notes

The standard `wails build` command produces an ar archive instead of a Mach-O executable when run from the TRAE sandbox environment. This is because the sandbox restricts CGO linking. The workaround is to use `scripts/build.sh`, which:

1. Builds the frontend with `pnpm --dir frontend build`
2. Compiles the Go binary with explicit `CGO_ENABLED=1` and `CGO_LDFLAGS="-framework UniformTypeIdentifiers"`
3. Packages the binary into a `.app` bundle with `Info.plist`
4. Signs the bundle with `codesign --force --deep --sign -`

The `wails dev` command also requires `-skipbindings` in the sandbox environment because the binding generator binary cannot be executed from `/var/folders/`.
