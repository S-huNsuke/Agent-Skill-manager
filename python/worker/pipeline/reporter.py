from __future__ import annotations

import json
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from providers.base import BaseProvider

_SYSTEM_PROMPT = """你是一个技能管理报告助手。你会收到执行结果，需要生成用户友好的摘要报告。

你必须返回纯 JSON（不要 markdown 代码块），格式如下：
{
  "status": "completed|failed|blocked",
  "summary": "用一两句话总结执行结果",
  "details": "详细说明，包括成功和失败的具体信息",
  "recommendation": "下一步建议（如果有）"
}

规则：
- summary 和 details 必须使用中文
- recommendation 如果没有建议则为空字符串"""


def render_report(result: dict, provider: BaseProvider | None = None) -> dict:
    """根据执行结果生成报告，可选使用 LLM 提供者"""
    status = result.get("status", "unknown")
    fallback = {
        "status": status,
        "summary": f"管线状态：{status}",
        "result": result,
    }

    if provider is None:
        return fallback

    import asyncio

    from providers.base import ProviderRequest

    request = ProviderRequest(
        system_prompt=_SYSTEM_PROMPT,
        user_prompt=f"执行结果：{json.dumps(result, ensure_ascii=False)}",
    )

    try:
        response = asyncio.run(provider.complete(request))
        report = json.loads(response.text)
        report.setdefault("result", result)
        return report
    except Exception:
        return fallback
