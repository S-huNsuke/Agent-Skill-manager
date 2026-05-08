from abc import ABC, abstractmethod
from dataclasses import dataclass


@dataclass
class ProviderRequest:
    """LLM 调用请求"""
    system_prompt: str
    user_prompt: str


@dataclass
class ProviderResponse:
    """LLM 调用响应"""
    text: str


class BaseProvider(ABC):
    """LLM 提供者抽象基类"""

    @abstractmethod
    async def complete(self, request: ProviderRequest) -> ProviderResponse:
        """发送补全请求并返回响应"""
        ...
