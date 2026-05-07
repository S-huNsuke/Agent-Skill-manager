import argparse
import json
import sys

from pipeline.planner import make_plan
from pipeline.reporter import render_report
from pipeline.resolver import resolve_plan
from providers.base import BaseProvider


def _create_provider(name: str, model: str | None) -> BaseProvider | None:
    """根据名称创建 LLM 提供者实例"""
    if name == "openai":
        from providers.openai_provider import OpenAIProvider

        return OpenAIProvider(model=model or "gpt-4o")
    if name == "anthropic":
        from providers.anthropic_provider import AnthropicProvider

        return AnthropicProvider(model=model or "claude-sonnet-4-20250514")
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

    return {"status": "error", "error": f"unsupported action: {action}"}


def main() -> int:
    parser = argparse.ArgumentParser(description="Agent Skills Manager AI Worker")
    parser.add_argument("--provider", default="none", help="LLM provider: openai, anthropic, or none")
    parser.add_argument("--model", default=None, help="Model name for the selected provider")
    args = parser.parse_args()

    provider = _create_provider(args.provider, args.model)

    payload = json.load(sys.stdin)
    json.dump(handle(payload, provider=provider), sys.stdout)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
