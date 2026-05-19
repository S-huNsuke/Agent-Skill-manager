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


def _create_provider(name: str, model: str | None, api_key: str | None = None, base_url: str | None = None) -> BaseProvider | None:
    """根据名称创建 LLM 提供者实例"""
    if name == "openai":
        from .providers.openai_provider import OpenAIProvider

        return OpenAIProvider(
            model=model or "gpt-4o",
            api_key=api_key or _env("ASM_AI_API_KEY"),
            base_url=base_url or _env("ASM_AI_BASE_URL") or None,
        )
    if name == "anthropic":
        from .providers.anthropic_provider import AnthropicProvider

        return AnthropicProvider(
            model=model or "claude-sonnet-4-20250514",
            api_key=api_key or _env("ASM_AI_API_KEY"),
            base_url=base_url or _env("ASM_AI_BASE_URL") or None,
        )
    if name == "gemini":
        from .providers.gemini_provider import GeminiProvider

        return GeminiProvider(
            model=model or "gemini-2.0-flash",
            api_key=api_key or _env("ASM_AI_API_KEY"),
            base_url=base_url or _env("ASM_AI_BASE_URL") or None,
        )
    if name == "openai-compatible":
        from .providers.openai_provider import OpenAIProvider

        return OpenAIProvider(
            model=model or "gpt-4o",
            api_key=api_key or _env("ASM_AI_API_KEY"),
            base_url=base_url or _env("ASM_AI_BASE_URL") or None,
        )
    return None


def _payload_runtime_config(payload: dict, args: argparse.Namespace) -> tuple[str, str | None, str | None, str | None]:
    body = payload.get("payload", {})
    config = body.get("config", {}) if isinstance(body, dict) else {}
    if not isinstance(config, dict):
        config = {}

    provider = str(config.get("provider") or args.provider or _env("ASM_AI_PROVIDER") or "none")
    model_value = config.get("model") or args.model or _env("ASM_AI_MODEL")
    api_key_value = config.get("api_key") or _env("ASM_AI_API_KEY")
    base_url_value = config.get("base_url") or _env("ASM_AI_BASE_URL")

    model = str(model_value) if model_value else None
    api_key = str(api_key_value) if api_key_value else None
    base_url = str(base_url_value) if base_url_value else None
    return provider, model, api_key, base_url


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

    try:
        payload = json.load(sys.stdin)
        provider_name, model, api_key, base_url = _payload_runtime_config(payload, args)
        provider = _create_provider(provider_name, model, api_key=api_key, base_url=base_url)
        response = handle(payload, provider=provider)
    except Exception as exc:  # pragma: no cover - defensive worker boundary
        response = {"status": "error", "data": {"error": str(exc)}}

    json.dump(response, sys.stdout)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
