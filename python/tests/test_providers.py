from __future__ import annotations

import json
import sys
from pathlib import Path
from unittest.mock import MagicMock, patch

ROOT = Path(__file__).resolve().parents[2]
PYTHON_ROOT = ROOT / "python"
WORKER_ROOT = PYTHON_ROOT / "worker"
sys.path.insert(0, str(PYTHON_ROOT))

from worker.pipeline.chat import sanitize_reply
from worker.providers.anthropic_provider import AnthropicProvider
from worker.providers.base import ProviderRequest
from worker.providers.gemini_provider import GeminiProvider
from worker.providers.http_client import post_json
from worker.providers.openai_provider import OpenAIProvider


class TestPostJson:
    """http_client.post_json 单元测试"""

    @patch("worker.providers.http_client.request.urlopen")
    def test_post_json_returns_parsed_body(self, mock_urlopen: MagicMock) -> None:
        """正常响应应返回解析后的 JSON 字典"""
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps({"choices": [{"message": {"content": "hi"}}]}).encode()
        mock_resp.__enter__ = MagicMock(return_value=mock_resp)
        mock_resp.__exit__ = MagicMock(return_value=False)
        mock_urlopen.return_value = mock_resp

        result = post_json("https://api.example.com/v1/chat", {"Authorization": "Bearer k"}, {"model": "gpt"})

        assert result["choices"][0]["message"]["content"] == "hi"
        mock_urlopen.assert_called_once()

    @patch("worker.providers.http_client.request.urlopen")
    def test_post_json_raises_on_http_error(self, mock_urlopen: MagicMock) -> None:
        """HTTP 错误应抛出 RuntimeError 并包含状态码"""
        from io import BytesIO
        from urllib import error

        fp = BytesIO(b"rate limited")
        mock_urlopen.side_effect = error.HTTPError(
            url="https://api.example.com", code=429, msg="Too Many Requests", hdrs=None, fp=fp
        )

        try:
            post_json("https://api.example.com/v1/chat", {}, {})
        except RuntimeError as exc:
            assert "429" in str(exc)
        else:
            raise AssertionError("Expected RuntimeError")

    @patch("worker.providers.http_client.request.urlopen")
    def test_post_json_raises_on_url_error(self, mock_urlopen: MagicMock) -> None:
        """URL 错误应抛出 RuntimeError"""
        from urllib import error

        mock_urlopen.side_effect = error.URLError(reason="connection refused")

        try:
            post_json("https://api.example.com/v1/chat", {}, {})
        except RuntimeError as exc:
            assert "connection refused" in str(exc)
        else:
            raise AssertionError("Expected RuntimeError")

    @patch("worker.providers.http_client.request.urlopen")
    def test_post_json_returns_empty_dict_on_empty_body(self, mock_urlopen: MagicMock) -> None:
        """空响应体应返回空字典"""
        mock_resp = MagicMock()
        mock_resp.read.return_value = b""
        mock_resp.__enter__ = MagicMock(return_value=mock_resp)
        mock_resp.__exit__ = MagicMock(return_value=False)
        mock_urlopen.return_value = mock_resp

        result = post_json("https://api.example.com/v1/chat", {}, {})

        assert result == {}


class TestOpenAIProviderRequest:
    """OpenAI Provider 请求构造测试"""

    def test_ignores_provider_specific_env_fallbacks(self) -> None:
        """应只读取 ASM_AI_*，不回退到供应商专用环境变量"""
        with patch.dict("os.environ", {"OPENAI_API_KEY": "provider-secret", "OPENAI_BASE_URL": "https://provider.example/v1"}, clear=True):
            provider = OpenAIProvider()
        assert provider.api_key == ""
        assert provider.base_url == "https://api.openai.com/v1"

    def test_builds_correct_headers(self) -> None:
        """验证请求头包含 Authorization 和 Content-Type"""
        provider = OpenAIProvider(api_key="test-key")
        assert provider.api_key == "test-key"
        assert provider.base_url == "https://api.openai.com/v1"

    def test_custom_base_url(self) -> None:
        """自定义 base_url 应正确设置"""
        provider = OpenAIProvider(base_url="https://custom.api.com/v1")
        assert provider.base_url == "https://custom.api.com/v1"

    def test_default_model(self) -> None:
        """默认模型应为 gpt-4o"""
        provider = OpenAIProvider()
        assert provider.model == "gpt-4o"

    @patch("worker.providers.openai_provider.post_json")
    def test_complete_sends_correct_payload(self, mock_post: MagicMock) -> None:
        """complete 方法应发送正确的请求结构"""
        mock_post.return_value = {"choices": [{"message": {"content": "response"}}]}
        provider = OpenAIProvider(api_key="k", model="gpt-4o-mini", max_tokens=1024)

        import asyncio

        result = asyncio.run(provider.complete(ProviderRequest(system_prompt="sys", user_prompt="usr")))

        assert result.text == "response"
        call_args = mock_post.call_args
        url = call_args[0][0]
        headers = call_args[0][1]
        payload = call_args[0][2]

        assert url.endswith("/chat/completions")
        assert headers["Authorization"] == "Bearer k"
        assert payload["model"] == "gpt-4o-mini"
        assert payload["max_tokens"] == 1024
        assert payload["messages"][0]["role"] == "system"
        assert payload["messages"][1]["role"] == "user"


class TestAnthropicProviderRequest:
    """Anthropic Provider 请求构造测试"""

    def test_builds_correct_headers(self) -> None:
        """验证请求头包含 x-api-key 和 anthropic-version"""
        provider = AnthropicProvider(api_key="sk-ant-test")
        assert provider.api_key == "sk-ant-test"
        assert provider.base_url == "https://api.anthropic.com"

    def test_default_model(self) -> None:
        """默认模型应为 claude-sonnet-4-20250514"""
        provider = AnthropicProvider()
        assert provider.model == "claude-sonnet-4-20250514"

    @patch("worker.providers.anthropic_provider.post_json")
    def test_complete_sends_correct_payload(self, mock_post: MagicMock) -> None:
        """complete 方法应发送正确的请求结构"""
        mock_post.return_value = {"content": [{"type": "text", "text": "hello"}]}
        provider = AnthropicProvider(api_key="k")

        import asyncio

        result = asyncio.run(provider.complete(ProviderRequest(system_prompt="sys", user_prompt="usr")))

        assert result.text == "hello"
        call_args = mock_post.call_args
        url = call_args[0][0]
        headers = call_args[0][1]
        payload = call_args[0][2]

        assert url.endswith("/v1/messages")
        assert headers["x-api-key"] == "k"
        assert headers["anthropic-version"] == "2023-06-01"
        assert payload["system"] == "sys"
        assert payload["messages"][0]["role"] == "user"


class TestGeminiProviderRequest:
    """Gemini Provider 请求构造测试"""

    def test_builds_correct_config(self) -> None:
        """验证 API key 和 base_url 配置"""
        provider = GeminiProvider(api_key="gem-key")
        assert provider.api_key == "gem-key"
        assert provider.base_url == "https://generativelanguage.googleapis.com/v1beta"

    def test_default_model(self) -> None:
        """默认模型应为 gemini-2.0-flash"""
        provider = GeminiProvider()
        assert provider.model == "gemini-2.0-flash"

    @patch("worker.providers.gemini_provider.post_json")
    def test_complete_sends_correct_payload(self, mock_post: MagicMock) -> None:
        """complete 方法应发送正确的请求结构"""
        mock_post.return_value = {
            "candidates": [{"content": {"parts": [{"text": "world"}]}}]
        }
        provider = GeminiProvider(api_key="k")

        import asyncio

        result = asyncio.run(provider.complete(ProviderRequest(system_prompt="sys", user_prompt="usr")))

        assert result.text == "world"
        call_args = mock_post.call_args
        url = call_args[0][0]
        headers = call_args[0][1]
        payload = call_args[0][2]

        assert ":generateContent" in url
        assert headers["x-goog-api-key"] == "k"
        assert payload["system_instruction"]["parts"][0]["text"] == "sys"
        assert payload["contents"][0]["parts"][0]["text"] == "usr"


class TestSanitizeReply:
    """sanitize_reply 边界条件测试"""

    def test_removes_think_tags(self) -> None:
        """应移除 <think/> 标签内容"""
        result = sanitize_reply("<think>internal reasoning</think>\nfinal answer")
        assert "internal reasoning" not in result
        assert "final answer" in result

    def test_removes_thinking_tags(self) -> None:
        """应移除 <thinking/> 标签内容"""
        result = sanitize_reply("<thinking>deep thought</thinking>answer")
        assert "deep thought" not in result
        assert "answer" in result

    def test_removes_code_block_thinking(self) -> None:
        """应移除 ```thinking 代码块"""
        result = sanitize_reply("```thinking\nsome reasoning\n```\nactual reply")
        assert "some reasoning" not in result
        assert "actual reply" in result

    def test_removes_chinese_thinking_prefix(self) -> None:
        """应移除中文思考过程前缀"""
        result = sanitize_reply("思考过程：这是推理\n\n最终答案：这是回答")
        assert "这是推理" not in result
        assert "这是回答" in result

    def test_strips_whitespace(self) -> None:
        """应去除首尾空白"""
        result = sanitize_reply("  hello world  ")
        assert result == "hello world"

    def test_empty_string(self) -> None:
        """空字符串应返回空字符串"""
        result = sanitize_reply("")
        assert result == ""

    def test_no_thinking_tags(self) -> None:
        """无思考标签的文本应原样返回"""
        text = "这是一条普通的回复，没有任何思考标签。"
        result = sanitize_reply(text)
        assert result == text

    def test_nested_think_tags(self) -> None:
        """嵌套思考标签应只移除匹配的内容"""
        result = sanitize_reply("<think>first</think>rest")
        assert "first" not in result
        assert "rest" in result

    def test_removes_reasoning_tags(self) -> None:
        """应移除 <reasoning/> 标签"""
        result = sanitize_reply("<reasoning>logic</reasoning>conclusion")
        assert "logic" not in result
        assert "conclusion" in result

    def test_removes_analysis_tags(self) -> None:
        """应移除 <analysis/> 标签"""
        result = sanitize_reply("<analysis>breakdown</analysis>result")
        assert "breakdown" not in result
        assert "result" in result

    def test_case_insensitive_tag_removal(self) -> None:
        """标签移除应不区分大小写"""
        result = sanitize_reply("<THINK>secret</THINK>public")
        assert "secret" not in result
        assert "public" in result

    def test_multiline_think_content(self) -> None:
        """多行思考内容应被完整移除"""
        text = "<think>\nline1\nline2\nline3\n</think>\noutput"
        result = sanitize_reply(text)
        assert "line1" not in result
        assert "line2" not in result
        assert "line3" not in result
        assert "output" in result
