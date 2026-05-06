import os

from .base import BaseProvider, ProviderRequest, ProviderResponse


class OpenAIProvider(BaseProvider):
    """基于 OpenAI Chat Completions API 的提供者"""

    def __init__(self, model: str = "gpt-4o", api_key: str | None = None, base_url: str | None = None, max_tokens: int = 2048):
        self.model = model
        self.api_key = api_key or os.environ.get("OPENAI_API_KEY", "")
        self.base_url = base_url or "https://api.openai.com/v1"
        self.max_tokens = max_tokens

    async def complete(self, request: ProviderRequest) -> ProviderResponse:
        """调用 OpenAI Chat Completions API"""
        import httpx

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

        async with httpx.AsyncClient(timeout=60.0) as client:
            response = await client.post(
                f"{self.base_url}/chat/completions",
                headers=headers,
                json=payload,
            )
            response.raise_for_status()
            data = response.json()

        text = data["choices"][0]["message"]["content"]
        return ProviderResponse(text=text)
