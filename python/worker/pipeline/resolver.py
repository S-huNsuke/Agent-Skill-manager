from __future__ import annotations

import json
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from providers.base import BaseProvider

_SYSTEM_PROMPT = """你是一个技能依赖解析助手。你会收到一个执行计划和当前环境状态，需要解析出具体的操作步骤。

你必须返回纯 JSON（不要 markdown 代码块），格式如下：
{
  "goal": "原始目标",
  "steps": [...原始步骤...],
  "prerequisites": {
    "has_artifact": true/false,
    "adapter_owns_target": true/false
  },
  "status": "ready|blocked",
  "actions": [
    {"skill_id": "技能ID", "version": "版本号", "target_agent": "目标适配器", "action": "install|update|repair"}
  ]
}

规则：
- 如果环境满足所有前提条件，status 为 ready，否则为 blocked
- actions 数组列出需要执行的具体操作
- 所有文本字段使用中文"""


def resolve_plan(
    plan: dict,
    *,
    has_artifact: bool,
    adapter_owns_target: bool,
    provider: BaseProvider | None = None,
) -> dict:
    """解析执行计划，确定具体操作和状态，可选使用 LLM 提供者"""
    resolved = dict(plan)
    resolved["prerequisites"] = {
        "has_artifact": has_artifact,
        "adapter_owns_target": adapter_owns_target,
    }
    resolved["status"] = "ready" if has_artifact and adapter_owns_target else "blocked"

    if provider is None:
        return resolved

    import asyncio

    from providers.base import ProviderRequest

    env_state = {
        "has_artifact": has_artifact,
        "adapter_owns_target": adapter_owns_target,
    }

    request = ProviderRequest(
        system_prompt=_SYSTEM_PROMPT,
        user_prompt=f"执行计划：{json.dumps(plan, ensure_ascii=False)}\n\n环境状态：{json.dumps(env_state, ensure_ascii=False)}",
    )

    try:
        response = asyncio.run(provider.complete(request))
        llm_result = json.loads(response.text)
        if "status" not in llm_result:
            raise ValueError("missing status field")
        resolved.update(llm_result)
        return resolved
    except Exception:
        return resolved
