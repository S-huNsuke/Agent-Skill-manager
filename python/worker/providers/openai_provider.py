from __future__ import annotations

import asyncio
import os

from .base import BaseProvider, ProviderRequest, ProviderResponse
from .http_client import post_json


class OpenAIProvider(BaseProvider):
    """基于 OpenAI Chat Completions API 的提供者"""

    def __init__(self, model: str = "gpt-4o", api_key: str | None = None, base_url: str | None = None, max_tokens: int = 2048):
        self.model = model
        self.api_key = api_key or os.environ.get("ASM_AI_API_KEY", "")
        self.base_url = (base_url or os.environ.get("ASM_AI_BASE_URL") or "https://api.openai.com/v1").rstrip("/")
        self.max_tokens = max_tokens

    async def complete(self, request: ProviderRequest) -> ProviderResponse:
        """调用 OpenAI Chat Completions API"""
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json",
        }
        payload = {
            "model": self.model,
            "messages": [
                {"role": "system", "content": request.system_prompt},
                {"role": "user", "content": request.user_prompt},
            ],
            "max_tokens": self.max_tokens,
        }

        data = await asyncio.to_thread(
            post_json,
            f"{self.base_url}/chat/completions",
            headers,
            payload,
        )

        text = data["choices"][0]["message"]["content"]
        return ProviderResponse(text=text)
