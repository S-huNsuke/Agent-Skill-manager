# Project Architecture Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Clean generated artifacts, remove unused code, and split oversized modules so the project structure is easier to maintain.

**Architecture:** Keep runtime behavior unchanged while improving boundaries. Frontend keeps `AssistantPanel` as the single assistant UI; Go app bindings are split by feature area while staying in package `app`; generated/cache files are excluded from source control.

**Tech Stack:** Go/Wails, React/TypeScript/Vite, SQLite, Python worker.

---

### Task 1: Cleanup Rules and Generated Files

**Files:**
- Modify: `.gitignore`
- Delete from workspace if present: `.gocache/`, `.pytest_cache/`, Python `__pycache__/`, `frontend/tsconfig.tsbuildinfo`, `test-screenshots/`, `build/tmp/`

- [ ] Add ignore entries for generated frontend, cache, and screenshot files.
- [ ] Remove generated/cache files from the working tree.
- [ ] Verify `git status --short` no longer shows generated cache artifacts.

### Task 2: Remove Dead Assistant Page

**Files:**
- Delete: `frontend/src/features/assistant/AssistantPage.tsx`

- [ ] Confirm no runtime import references `AssistantPage`.
- [ ] Delete the unused page file.
- [ ] Run TypeScript check to verify no missing import.

### Task 3: Normalize Build Scripts

**Files:**
- Create: `scripts/build-and-run.sh`
- Modify: `script/build_and_run.sh`

- [ ] Move the implementation to `scripts/build-and-run.sh`.
- [ ] Keep `script/build_and_run.sh` as a compatibility wrapper.
- [ ] Verify `./script/build_and_run.sh --verify` still works.

### Task 4: Split Frontend Types and Mock Data

**Files:**
- Create: `frontend/src/lib/types.ts`
- Create: `frontend/src/lib/mockData.ts`
- Modify: `frontend/src/lib/mocks.ts`
- Modify imports only if needed after the compatibility file is in place.

- [ ] Move exported interfaces/types to `types.ts`.
- [ ] Move `mockSnapshot` data to `mockData.ts`.
- [ ] Keep `mocks.ts` as a compatibility re-export to avoid a large frontend import churn in this pass.
- [ ] Run frontend tests and typecheck.

### Task 5: Split Go App Bindings by Responsibility

**Files:**
- Modify: `internal/app/bindings.go`
- Create: `internal/app/bindings_assistant.go`
- Create: `internal/app/bindings_settings.go`
- Create: `internal/app/bindings_catalog.go`
- Create: `internal/app/bindings_projects.go`
- Create: `internal/app/bindings_skills.go`

- [ ] Move assistant functions and helpers into `bindings_assistant.go`.
- [ ] Move settings/log/app info functions into `bindings_settings.go`.
- [ ] Move catalog/store functions and GitHub helpers into `bindings_catalog.go`.
- [ ] Move project/reconcile functions into `bindings_projects.go`.
- [ ] Move skill/agent functions into `bindings_skills.go`.
- [ ] Keep package name `app` and preserve public method signatures for Wails.
- [ ] Run `gofmt` and full Go tests.

### Task 6: Final Verification

- [ ] Run `pnpm --dir frontend exec tsc --noEmit`.
- [ ] Run `pnpm --dir frontend test -- --run`.
- [ ] Run `GOCACHE="$PWD/.gocache" go test ./...`.
- [ ] Run `./script/build_and_run.sh --verify`.
- [ ] Summarize changed architecture and any remaining risks.
