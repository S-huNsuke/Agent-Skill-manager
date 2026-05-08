from __future__ import annotations

import asyncio
import re
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from worker.providers.base import BaseProvider

_SYSTEM_PROMPT = """你是 Agent Skills Manager 内置的通用 AI 聊天助手。

你可以自然地回答用户问题，也可以在用户询问技能、代理、配置或环境问题时给出具体步骤。
请直接回答用户，不要返回 JSON，不要只生成执行计划。
如果用户的问题和本应用能力有关，优先给出可执行的简短建议。
"""


def sanitize_reply(text: str) -> str:
    """移除推理模型可能返回的内部思考内容。"""
    cleaned = text.strip()
    cleaned = re.sub(r"<\s*(think|thinking|reasoning|analysis)\s*>.*?<\s*/\s*\1\s*>", "", cleaned, flags=re.IGNORECASE | re.DOTALL)
    cleaned = re.sub(r"^```(?:think|thinking|reasoning|analysis)\s+.*?```\s*", "", cleaned, flags=re.IGNORECASE | re.DOTALL)
    cleaned = re.sub(
        r"^\s*(?:思考过程|思考|推理过程|推理|Thinking|Reasoning|Analysis)\s*[:：].*?(?:\n\s*\n|(?=最终答案\s*[:：])|(?=回答\s*[:：]))",
        "",
        cleaned,
        flags=re.IGNORECASE | re.DOTALL,
    )
    cleaned = re.sub(r"^\s*(?:最终答案|回答)\s*[:：]\s*", "", cleaned)
    return cleaned.strip()


def chat(message: str, history: list[dict] | None = None, provider: BaseProvider | None = None) -> dict:
    """返回普通聊天回复。没有 provider 时使用本地可解释 fallback。"""
    trimmed = message.strip()
    if provider is None:
        return {
            "reply": f"我收到你的消息了：{trimmed}\n\n当前没有启用外部 AI 模型，所以只能使用本地模式。请在 AI 配置里选择供应商、填写 API Key 和模型后，我就能像聊天机器人一样直接回答。",
            "provider": "none",
            "model": "",
        }

    from worker.providers.base import ProviderRequest

    history_lines: list[str] = []
    for item in (history or [])[-12:]:
        role = str(item.get("role", "user"))
        content = str(item.get("content", "")).strip()
        if content:
            history_lines.append(f"{role}: {content}")

    user_prompt = trimmed
    if history_lines:
        user_prompt = "最近对话：\n" + "\n".join(history_lines) + "\n\n用户新消息：\n" + trimmed

    try:
        response = asyncio.run(provider.complete(ProviderRequest(system_prompt=_SYSTEM_PROMPT, user_prompt=user_prompt)))
        reply = sanitize_reply(response.text)
        return {
            "reply": reply or "模型返回了空内容。",
        }
    except Exception as exc:
        return {
            "reply": f"AI 请求失败：{exc}",
            "error": str(exc),
        }
