from __future__ import annotations

import json
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from worker.providers.base import BaseProvider

_FALLBACK_STEPS = [
    {"action": "recommend", "label": "推荐技能", "detail": "根据目标推荐合适的技能"},
    {"action": "resolve", "label": "解析依赖", "detail": "解析技能依赖和兼容性"},
    {"action": "execute", "label": "执行安装", "detail": "安装或修复技能"},
    {"action": "verify", "label": "验证结果", "detail": "验证安装结果和完整性"},
]

_SYSTEM_PROMPT = """你是一个技能管理规划助手。用户会给你一个目标，你需要返回一个执行计划。

你会收到当前系统的状态信息，包括：
- 已安装的代理（agents）和技能（skills）
- 可用的商店技能（available_skills）
- 项目信息

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
- revision 从 1 开始
- 根据系统当前状态（已安装的技能、代理等）给出合理的建议
- 如果用户询问技能相关问题，优先从 available_skills 中推荐"""


def make_plan(goal: str, context: dict | None = None, provider: BaseProvider | None = None) -> dict:
    """根据目标生成执行计划，可选使用 LLM 提供者"""
    def with_context(plan: dict) -> dict:
        """在计划中保留解析阶段需要的上下文。"""
        if context:
            plan.setdefault("context", context)
        return plan

    if provider is None:
        return with_context({"goal": goal, "steps": list(_FALLBACK_STEPS), "revision": 1})

    import asyncio

    from worker.providers.base import ProviderRequest

    # 构建包含上下文的用户提示
    user_prompt = f"目标：{goal}"

    if context:
        user_prompt += "\n\n当前系统状态："
        user_prompt += f"\n- 已安装代理数量：{context.get('agents_count', 0)}"
        user_prompt += f"\n- 已安装技能数量：{context.get('skills_count', 0)}"
        user_prompt += f"\n- 项目数量：{context.get('projects_count', 0)}"

        # 添加已安装的技能列表
        skills = context.get('skills', [])
        if skills:
            user_prompt += "\n\n已安装的技能："
            for skill in skills[:10]:  # 只显示前10个
                user_prompt += f"\n- {skill.get('name')} ({skill.get('agent')})：{skill.get('summary', '')}"

        # 添加可用的商店技能
        available_skills = context.get('available_skills', [])
        if available_skills:
            user_prompt += "\n\n可用的商店技能（部分）："
            for skill in available_skills[:10]:  # 只显示前10个
                user_prompt += f"\n- {skill.get('name')} by {skill.get('author')}: {skill.get('summary', '')[:80]}"

    request = ProviderRequest(
        system_prompt=_SYSTEM_PROMPT,
        user_prompt=user_prompt,
    )

    try:
        response = asyncio.run(provider.complete(request))
        plan = json.loads(response.text)
        if "goal" not in plan or "steps" not in plan:
            raise ValueError("missing required fields")
        plan.setdefault("revision", 1)
        return with_context(plan)
    except Exception:
        return with_context({"goal": goal, "steps": list(_FALLBACK_STEPS), "revision": 1})
