from __future__ import annotations

import json
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from providers.base import BaseProvider

_FALLBACK_STEPS = [
    {"action": "recommend", "label": "推荐技能", "detail": "根据目标推荐合适的技能"},
    {"action": "resolve", "label": "解析依赖", "detail": "解析技能依赖和兼容性"},
    {"action": "execute", "label": "执行安装", "detail": "安装或修复技能"},
    {"action": "verify", "label": "验证结果", "detail": "验证安装结果和完整性"},
]

_SYSTEM_PROMPT = """你是一个技能管理规划助手。用户会给你一个目标，你需要返回一个执行计划。

你必须返回纯 JSON（不要 markdown 代码块），格式如下：
{
  "goal": "用户的目标",
  "steps": [
    {"action": "recommend|resolve|execute|verify|repair", "label": "步骤名称", "detail": "步骤说明"}
  ],
  "revision": 1
}

规则：
- steps 数组至少包含 2 个步骤
- action 只能是 recommend、resolve、execute、verify、repair 之一
- label 和 detail 必须是中文
- revision 从 1 开始"""


def make_plan(goal: str, provider: BaseProvider | None = None) -> dict:
    """根据目标生成执行计划，可选使用 LLM 提供者"""
    if provider is None:
        return {"goal": goal, "steps": list(_FALLBACK_STEPS), "revision": 1}

    import asyncio

    from providers.base import ProviderRequest

    request = ProviderRequest(
        system_prompt=_SYSTEM_PROMPT,
        user_prompt=f"目标：{goal}",
    )

    try:
        response = asyncio.run(provider.complete(request))
        plan = json.loads(response.text)
        if "goal" not in plan or "steps" not in plan:
            raise ValueError("missing required fields")
        plan.setdefault("revision", 1)
        return plan
    except Exception:
        return {"goal": goal, "steps": list(_FALLBACK_STEPS), "revision": 1}
