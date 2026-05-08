from __future__ import annotations

import argparse
import json
import os
import sys

from .pipeline.planner import make_plan
from .pipeline.reporter import render_report
from .pipeline.resolver import resolve_plan
from .pipeline.chat import chat
from .providers.base import BaseProvider


def _env(name: str, default: str = "") -> str:
    value = os.environ.get(name)
    if value is None:
        return default
    return value


def _create_provider(name: str, model: str | None) -> BaseProvider | None:
    """根据名称创建 LLM 提供者实例"""
    if name == "openai":
        from .providers.openai_provider import OpenAIProvider

        api_key = _env("ASM_AI_API_KEY") or _env("OPENAI_API_KEY")
        base_url = _env("ASM_AI_BASE_URL") or _env("OPENAI_BASE_URL")
        return OpenAIProvider(model=model or "gpt-4o", api_key=api_key, base_url=base_url or None)
    if name == "anthropic":
        from .providers.anthropic_provider import AnthropicProvider

        api_key = _env("ASM_AI_API_KEY") or _env("ANTHROPIC_API_KEY")
        base_url = _env("ASM_AI_BASE_URL") or _env("ANTHROPIC_BASE_URL")
        return AnthropicProvider(model=model or "claude-sonnet-4-20250514", api_key=api_key, base_url=base_url or None)
    if name == "gemini":
        from .providers.gemini_provider import GeminiProvider

        api_key = _env("ASM_AI_API_KEY") or _env("GEMINI_API_KEY")
        base_url = _env("ASM_AI_BASE_URL") or _env("GEMINI_BASE_URL")
        return GeminiProvider(model=model or "gemini-2.0-flash", api_key=api_key, base_url=base_url or None)
    if name == "openai-compatible":
        from .providers.openai_provider import OpenAIProvider

        api_key = _env("ASM_AI_API_KEY") or _env("OPENAI_API_KEY")
        base_url = _env("ASM_AI_BASE_URL") or _env("OPENAI_BASE_URL")
        return OpenAIProvider(model=model or "gpt-4o", api_key=api_key, base_url=base_url or None)
    return None


def handle(payload: dict, provider: BaseProvider | None = None) -> dict:
    """处理来自 Go 核心的请求，分派到对应的管线阶段"""
    action = payload.get("action")
    body = payload.get("payload", {})

    if action == "plan":
        context = body.get("context")
        return {"status": "ok", "data": make_plan(body["goal"], context=context, provider=provider)}
    if action == "resolve":
        return {
            "status": "ok",
            "data": resolve_plan(
                body["plan"],
                has_artifact=body.get("has_artifact", False),
                adapter_owns_target=body.get("adapter_owns_target", False),
                provider=provider,
            ),
        }
    if action == "report":
        return {"status": "ok", "data": render_report(body["result"], provider=provider)}
    if action == "chat":
        return {"status": "ok", "data": chat(body.get("message", ""), history=body.get("history", []), provider=provider)}

    return {"status": "error", "data": {"error": f"unsupported action: {action}"}}


def main() -> int:
    parser = argparse.ArgumentParser(description="Agent Skills Manager AI Worker")
    parser.add_argument("--provider", default="none", help="LLM provider: openai, anthropic, or none")
    parser.add_argument("--model", default=None, help="Model name for the selected provider")
    args = parser.parse_args()

    provider = _create_provider(args.provider, args.model)

    try:
        payload = json.load(sys.stdin)
        response = handle(payload, provider=provider)
    except Exception as exc:  # pragma: no cover - defensive worker boundary
        response = {"status": "error", "data": {"error": str(exc)}}

    json.dump(response, sys.stdout)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
