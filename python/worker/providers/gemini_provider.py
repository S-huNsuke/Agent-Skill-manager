from __future__ import annotations

import asyncio
import os

from .base import BaseProvider, ProviderRequest, ProviderResponse
from .http_client import post_json


class GeminiProvider(BaseProvider):
    """基于 Google Gemini generateContent API 的提供者"""

    def __init__(self, model: str = "gemini-2.0-flash", api_key: str | None = None, base_url: str | None = None, max_tokens: int = 2048):
        self.model = model
        self.api_key = api_key or os.environ.get("ASM_AI_API_KEY") or os.environ.get("GEMINI_API_KEY", "")
        self.base_url = (base_url or os.environ.get("ASM_AI_BASE_URL") or os.environ.get("GEMINI_BASE_URL") or "https://generativelanguage.googleapis.com/v1beta").rstrip("/")
        self.max_tokens = max_tokens

    async def complete(self, request: ProviderRequest) -> ProviderResponse:
        """调用 Gemini generateContent API"""
        headers = {
            "Content-Type": "application/json",
            "x-goog-api-key": self.api_key,
        }
        payload = {
            "system_instruction": {
                "parts": [{"text": request.system_prompt}],
            },
            "contents": [
                {
                    "role": "user",
                    "parts": [{"text": request.user_prompt}],
                }
            ],
            "generationConfig": {
                "maxOutputTokens": self.max_tokens,
                "temperature": 0.2,
            },
        }

        data = await asyncio.to_thread(
            post_json,
            f"{self.base_url}/models/{self.model}:generateContent",
            headers,
            payload,
        )

        parts = data["candidates"][0]["content"].get("parts", [])
        text = "".join(part.get("text", "") for part in parts)
        return ProviderResponse(text=text)
