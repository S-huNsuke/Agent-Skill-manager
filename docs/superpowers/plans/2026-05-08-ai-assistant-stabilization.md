# AI Assistant Stabilization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore the AI assistant to a working end-to-end flow and add configurable support for common API providers.

**Architecture:** Keep Go as the authoritative task orchestrator and Python as the LLM planning/resolution/reporting worker. Store the structured AI plan and action log in task records, pass provider credentials through the local worker process environment, and keep the frontend settings UI aligned with the backend provider schema.

**Tech Stack:** Go/Wails backend, SQLite persistence, Python worker with `httpx`, React/TypeScript frontend.

---

## Findings Summary

- Python worker startup is broken because Go runs `python3 -m worker.main` from `python/worker`, where the `worker` package is not importable.
- `resolve` receives `task.Records` as a string slice instead of the structured plan dict expected by Python.
- `report` receives `{goal, records}` while Python requires `result`, causing a `KeyError`.
- AI task persistence omits `project_id`, but the `tasks` table requires it and enforces a foreign key, so history writes can silently fail.
- API settings include API key and base URL in the UI, but only provider/model are applied to the bridge.
- The execution stage guesses skill names from text records instead of consuming structured resolved actions.
- The assistant UI has an unused full page and a working side panel; this plan stabilizes the side panel and keeps page cleanup out of scope unless needed for correctness.

## Provider Scope

Supported provider options:

- `none`: local deterministic fallback.
- `openai`: OpenAI Chat Completions.
- `anthropic`: Anthropic Messages API.
- `gemini`: Google Gemini generateContent API.
- `openai-compatible`: OpenAI-compatible Chat Completions endpoint, covering OpenRouter, DeepSeek, Azure-style gateways when the user supplies a compatible base URL and model.

## Implementation Tasks

### Task 1: Plan Document

**Files:**
- Create: `docs/superpowers/plans/2026-05-08-ai-assistant-stabilization.md`

- [x] **Step 1: Record discovered issues**

Write the findings above so the implementation can be checked against the original review.

- [x] **Step 2: Lock provider scope**

Use the approved provider scope: OpenAI, Anthropic, Google Gemini, OpenAI-compatible, and local fallback.

### Task 2: Python Worker Contract

**Files:**
- Modify: `python/worker/main.py`
- Modify: `python/worker/providers/openai_provider.py`
- Modify: `python/worker/providers/anthropic_provider.py`
- Create: `python/worker/providers/gemini_provider.py`
- Test: `python/tests/test_pipeline.py`

- [x] **Step 1: Add worker tests for `resolve` and `report` payloads**

Add tests that call `handle()` with the same payload shape Go will send:

```python
def test_main_handle_resolve_accepts_structured_plan() -> None:
    plan = {"goal": "install skill", "steps": [{"action": "recommend", "label": "推荐", "detail": "推荐技能"}], "revision": 1}
    response = handle({"action": "resolve", "payload": {"plan": plan, "has_artifact": True, "adapter_owns_target": True}})
    assert response["status"] == "ok"
    assert response["data"]["status"] == "ready"

def test_main_handle_report_accepts_result_payload() -> None:
    response = handle({"action": "report", "payload": {"result": {"status": "completed", "records": ["done"]}}})
    assert response["status"] == "ok"
    assert response["data"]["status"] == "completed"
```

- [x] **Step 2: Support Gemini and OpenAI-compatible providers**

Update provider creation to read API key/base URL from environment variables:

```python
ASM_AI_API_KEY
ASM_AI_BASE_URL
ASM_AI_PROVIDER
```

`openai-compatible` should reuse the OpenAI-compatible Chat Completions provider with a required custom base URL.

- [x] **Step 3: Make worker errors structured**

Wrap `handle()` in `main()` so unexpected worker exceptions emit:

```json
{"status":"error","data":{"error":"..."}}
```

instead of crashing without parseable stdout.

### Task 3: Go Bridge and Task Lifecycle

**Files:**
- Modify: `internal/ai/local_bridge.go`
- Modify: `internal/ai/bridge_test.go`
- Modify: `internal/app/viewmodels.go`
- Modify: `internal/app/app.go`
- Modify: `internal/app/bindings.go`
- Modify: `internal/app/bindings_test.go`

- [x] **Step 1: Add bridge configuration fields**

Extend `LocalBridge` with API key and base URL fields and pass them to Python as environment variables.

- [x] **Step 2: Fix worker startup path**

Run the module from `python`, not `python/worker`, so `worker.main` is importable.

- [x] **Step 3: Preserve structured plan/action state**

Add plan and resolved action fields to `AssistantTaskViewModel`, persist them to `Task.PlanJSON` and `Task.ActionLog`, and use them instead of parsing display records.

- [x] **Step 4: Fix report payload**

Call worker `report` with:

```go
Payload: map[string]any{"result": map[string]any{...}}
```

- [x] **Step 5: Fix task history persistence**

Either make task project IDs nullable or create/use a stable internal assistant project. Prefer a nullable migration for AI/system tasks to avoid fake project records.

### Task 4: Frontend AI Settings

**Files:**
- Modify: `frontend/src/lib/mocks.ts`
- Modify: `frontend/src/features/settings/SettingsPage.tsx`
- Modify: `frontend/src/features/assistant/AssistantPanel.tsx` if new task fields need display handling

- [x] **Step 1: Add provider choices**

Expose `none`, `openai`, `anthropic`, `gemini`, and `openai-compatible`.

- [x] **Step 2: Align model/base URL hints**

Use provider-specific placeholders so users understand which fields are required.

- [x] **Step 3: Keep API key local**

Continue saving settings through the local Wails binding. Do not log API keys.

### Task 5: Verification

**Commands:**

- [x] `python3 -m py_compile python/worker/main.py python/worker/pipeline/planner.py python/worker/pipeline/resolver.py python/worker/pipeline/reporter.py python/worker/providers/*.py`
- [x] Direct worker smoke tests for `plan`, `resolve`, and `report`.
- [x] `GOCACHE=/private/tmp/gocache go test ./...`
- [x] `pnpm --dir frontend exec tsc --noEmit`
- [x] `pnpm --dir frontend test -- --run`

Python `pytest` is optional in this environment because it is not installed. If available, run `python3 -m pytest python/tests -q`.

## Verification Results

- Go full test suite passed with `GOCACHE=/private/tmp/gocache go test ./...`.
- Frontend type checking passed with `pnpm --dir frontend exec tsc --noEmit`.
- Frontend Vitest suite passed with `pnpm --dir frontend test -- --run`.
- Python worker syntax checks passed with `python3 -m py_compile`.
- Python worker direct smoke checks passed for `plan`, `resolve`, and `report` via `python3 -m worker.main --provider none` from the `python` directory.
