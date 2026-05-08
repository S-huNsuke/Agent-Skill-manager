from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path
from unittest.mock import AsyncMock

ROOT = Path(__file__).resolve().parents[2]
PYTHON_ROOT = ROOT / "python"
WORKER_ROOT = PYTHON_ROOT / "worker"
sys.path.insert(0, str(PYTHON_ROOT))

from worker.main import handle
from worker.pipeline.chat import chat
from worker.pipeline.planner import make_plan
from worker.pipeline.reporter import render_report
from worker.pipeline.resolver import resolve_plan
from worker.providers.base import BaseProvider, ProviderRequest, ProviderResponse


def test_make_plan_returns_expected_steps() -> None:
    plan = make_plan("recommend skills for a Go project")

    assert plan["goal"] == "recommend skills for a Go project"
    assert len(plan["steps"]) >= 2
    assert plan["revision"] == 1


def test_make_plan_with_provider_returns_structured_plan() -> None:
    mock_provider = AsyncMock(spec=BaseProvider)
    mock_provider.complete.return_value = ProviderResponse(
        text=json.dumps({
            "goal": "test goal",
            "steps": [{"action": "recommend", "label": "推荐", "detail": "推荐技能"}],
            "revision": 1,
        })
    )

    plan = make_plan("test goal", provider=mock_provider)

    assert plan["goal"] == "test goal"
    assert len(plan["steps"]) >= 1
    mock_provider.complete.assert_awaited_once()


def test_make_plan_falls_back_on_provider_error() -> None:
    mock_provider = AsyncMock(spec=BaseProvider)
    mock_provider.complete.side_effect = RuntimeError("API error")

    plan = make_plan("test goal", provider=mock_provider)

    assert plan["goal"] == "test goal"
    assert len(plan["steps"]) >= 2


def test_resolve_plan_blocks_when_prerequisites_are_missing() -> None:
    resolved = resolve_plan(
        make_plan("repair broken skills"),
        has_artifact=False,
        adapter_owns_target=True,
    )

    assert resolved["status"] == "blocked"
    assert resolved["prerequisites"]["has_artifact"] is False


def test_resolve_plan_with_provider_returns_resolved() -> None:
    mock_provider = AsyncMock(spec=BaseProvider)
    mock_provider.complete.return_value = ProviderResponse(
        text=json.dumps({
            "status": "ready",
            "actions": [{"skill_id": "s1", "version": "1.0", "target_agent": "codex", "action": "install"}],
        })
    )

    plan = make_plan("install a skill")
    resolved = resolve_plan(plan, has_artifact=True, adapter_owns_target=True, provider=mock_provider)

    assert resolved["status"] == "ready"
    mock_provider.complete.assert_awaited_once()


def test_render_report_wraps_pipeline_result() -> None:
    report = render_report({"status": "ready", "steps": 4})

    assert report["status"] == "ready"
    assert "ready" in report["summary"]


def test_render_report_with_provider_returns_structured_report() -> None:
    mock_provider = AsyncMock(spec=BaseProvider)
    mock_provider.complete.return_value = ProviderResponse(
        text=json.dumps({
            "status": "completed",
            "summary": "安装成功完成",
            "details": "所有技能已安装",
            "recommendation": "",
        })
    )

    report = render_report({"status": "completed"}, provider=mock_provider)

    assert report["status"] == "completed"
    assert "安装" in report["summary"]
    mock_provider.complete.assert_awaited_once()


def test_main_entrypoint_handles_plan_action() -> None:
    payload = {
        "action": "plan",
        "payload": {"goal": "set up a project skill group"},
    }
    completed = subprocess.run(
        [sys.executable, "-m", "worker.main"],
        input=json.dumps(payload),
        text=True,
        capture_output=True,
        cwd=PYTHON_ROOT,
        check=True,
    )

    response = json.loads(completed.stdout)
    assert response["status"] == "ok"
    assert response["data"]["goal"] == "set up a project skill group"


def test_handle_accepts_structured_resolve_payload() -> None:
    plan = {
        "goal": "install a skill",
        "steps": [{"action": "recommend", "label": "推荐技能", "detail": "推荐适合的技能"}],
        "revision": 1,
    }

    response = handle(
        {
            "action": "resolve",
            "payload": {
                "plan": plan,
                "has_artifact": True,
                "adapter_owns_target": True,
            },
        }
    )

    assert response["status"] == "ok"
    assert response["data"]["goal"] == "install a skill"
    assert response["data"]["status"] == "ready"
    assert response["data"]["prerequisites"]["has_artifact"] is True


def test_handle_accepts_result_payload_for_report() -> None:
    response = handle(
        {
            "action": "report",
            "payload": {
                "result": {
                    "status": "completed",
                    "records": ["done"],
                }
            },
        }
    )

    assert response["status"] == "ok"
    assert response["data"]["status"] == "completed"


def test_handle_accepts_chat_payload() -> None:
    response = handle(
        {
            "action": "chat",
            "payload": {
                "message": "你好",
                "history": [],
            },
        }
    )

    assert response["status"] == "ok"
    assert "reply" in response["data"]
    assert "你好" in response["data"]["reply"]


def test_chat_strips_reasoning_blocks_from_provider_reply() -> None:
    mock_provider = AsyncMock(spec=BaseProvider)
    mock_provider.complete.return_value = ProviderResponse(
        text="<think>这里是内部推理，不应显示</think>\n最终答案"
    )

    response = chat("你好", provider=mock_provider)

    assert response["reply"] == "最终答案"
    assert "内部推理" not in response["reply"]


def test_provider_request_response_dataclass() -> None:
    req = ProviderRequest(system_prompt="sys", user_prompt="usr")
    assert req.system_prompt == "sys"
    assert req.user_prompt == "usr"

    resp = ProviderResponse(text="hello")
    assert resp.text == "hello"
