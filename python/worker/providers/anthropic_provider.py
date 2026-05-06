import os

from .base import BaseProvider, ProviderRequest, ProviderResponse


class AnthropicProvider(BaseProvider):
    """基于 Anthropic Messages API 的提供者"""

    def __init__(self, model: str = "claude-sonnet-4-20250514", api_key: str | None = None, max_tokens: int = 2048):
        self.model = model
        self.api_key = api_key or os.environ.get("ANTHROPIC_API_KEY", "")
        self.max_tokens = max_tokens

    async def complete(self, request: ProviderRequest) -> ProviderResponse:
        """调用 Anthropic Messages API"""
        import httpx

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

        async with httpx.AsyncClient(timeout=60.0) as client:
            response = await client.post(
                "https://api.anthropic.com/v1/messages",
                headers=headers,
                json=payload,
            )
            response.raise_for_status()
            data = response.json()

        text = data["content"][0]["text"]
        return ProviderResponse(text=text)
