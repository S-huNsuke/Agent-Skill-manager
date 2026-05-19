from __future__ import annotations

import json
import re
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from worker.providers.base import BaseProvider

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


def _text_blob(value: object) -> str:
    """把计划里的文本字段展平成可搜索字符串。"""
    if isinstance(value, dict):
        return " ".join(_text_blob(item) for item in value.values())
    if isinstance(value, list):
        return " ".join(_text_blob(item) for item in value)
    return str(value or "")


def _normalize_skill_name(name: object) -> str:
    """将技能名规整为适合模糊匹配的字符串。"""
    return re.sub(r"[^a-z0-9\u4e00-\u9fff]+", "", str(name or "").lower())


def _infer_action(intent_text: str) -> str:
    """根据目标文本推断 install/update/repair 动作。"""
    lowered = intent_text.lower()
    if any(token in lowered for token in ("修复", "repair", "fix")):
        return "repair"
    if any(token in lowered for token in ("更新", "升级", "update", "upgrade")):
        return "update"
    return "install"


def _fallback_actions(plan: dict) -> list[dict]:
    """从计划上下文里保守推断可执行动作。"""
    context = plan.get("context")
    if not isinstance(context, dict):
        context = {}

    available_skills = context.get("available_skills", [])
    if not isinstance(available_skills, list):
        available_skills = []

    intent_text = _text_blob({"goal": plan.get("goal"), "steps": plan.get("steps")})
    goal_text = _text_blob(plan.get("goal"))
    normalized_text = _normalize_skill_name(intent_text)
    action = _infer_action(goal_text or intent_text)
    actions: list[dict] = []
    seen: set[str] = set()

    for skill in available_skills:
        if not isinstance(skill, dict):
            continue
        name = str(skill.get("name") or "").strip()
        if not name:
            continue
        normalized_name = _normalize_skill_name(name)
        if not normalized_name or normalized_name not in normalized_text:
            continue
        if name in seen:
            continue
        seen.add(name)
        actions.append({
            "skill_id": name,
            "version": "latest",
            "target_agent": "",
            "action": action,
        })

    return actions


def _valid_actions(actions: object) -> list[dict]:
    """过滤并规范化 resolver 返回的执行动作。"""
    if not isinstance(actions, list):
        return []

    result: list[dict] = []
    for item in actions:
        if not isinstance(item, dict):
            continue
        action = str(item.get("action") or "").strip()
        skill_id = str(item.get("skill_id") or item.get("skillId") or "").strip()
        if action not in {"install", "update", "repair"}:
            continue
        if not skill_id and action != "repair":
            continue
        result.append({
            "skill_id": skill_id,
            "version": str(item.get("version") or "latest"),
            "target_agent": str(item.get("target_agent") or item.get("targetAgent") or ""),
            "action": action,
        })
    return result


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
    resolved["actions"] = _fallback_actions(resolved) if resolved["status"] == "ready" else []
    if resolved["status"] == "ready" and not resolved["actions"]:
        resolved["status"] = "blocked"
        resolved["summary"] = "未能从计划中定位可执行技能，请先选择要安装、更新或修复的技能。"

    if provider is None:
        return resolved

    import asyncio

    from worker.providers.base import ProviderRequest

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
        resolved["actions"] = _valid_actions(resolved.get("actions"))
        if resolved.get("status") == "ready" and not resolved["actions"]:
            resolved["actions"] = _fallback_actions(resolved)
        if resolved.get("status") == "ready" and not resolved["actions"]:
            resolved["status"] = "blocked"
            resolved["summary"] = "AI 没有返回可执行动作，请先选择要安装、更新或修复的技能。"
        return resolved
    except Exception:
        return resolved
