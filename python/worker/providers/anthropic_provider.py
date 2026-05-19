from __future__ import annotations

import asyncio
import os

from .base import BaseProvider, ProviderRequest, ProviderResponse
from .http_client import post_json


class AnthropicProvider(BaseProvider):
    """基于 Anthropic Messages API 的提供者"""

    def __init__(self, model: str = "claude-sonnet-4-20250514", api_key: str | None = None, base_url: str | None = None, max_tokens: int = 2048):
        self.model = model
        self.api_key = api_key or os.environ.get("ASM_AI_API_KEY", "")
        self.base_url = (base_url or os.environ.get("ASM_AI_BASE_URL") or "https://api.anthropic.com").rstrip("/")
        self.max_tokens = max_tokens

    async def complete(self, request: ProviderRequest) -> ProviderResponse:
        """调用 Anthropic Messages API"""
        headers = {
            "x-api-key": self.api_key,
            "anthropic-version": "2023-06-01",
            "Content-Type": "application/json",
        }
        payload = {
            "model": self.model,
            "system": request.system_prompt,
            "messages": [
                {"role": "user", "content": request.user_prompt},
            ],
            "max_tokens": self.max_tokens,
        }

        data = await asyncio.to_thread(
            post_json,
            f"{self.base_url}/v1/messages",
            headers,
            payload,
        )

        content = data["content"][0]
        text = content["text"] if isinstance(content, dict) else str(content)
        return ProviderResponse(text=text)
