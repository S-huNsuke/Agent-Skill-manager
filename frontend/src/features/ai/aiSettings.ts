import type { AISettingsViewModel } from "../../lib/mocks";

export type AiPreset = "none" | "openai" | "anthropic" | "gemini" | "openrouter" | "deepseek" | "moonshot" | "qwen" | "zhipu" | "minimax" | "ollama" | "custom-compatible";

export const defaultAISettings: AISettingsViewModel = { provider: "none", model: "", apiKey: "", baseUrl: "" };

export const aiProviderPresets: Array<{
  value: AiPreset;
  label: string;
  provider: AISettingsViewModel["provider"];
  model: string;
  baseUrl: string;
}> = [
  { value: "none", label: "本地回退", provider: "none", model: "", baseUrl: "" },
  { value: "openai", label: "OpenAI 官方", provider: "openai", model: "gpt-4.1-mini", baseUrl: "" },
  { value: "anthropic", label: "Anthropic 官方", provider: "anthropic", model: "claude-sonnet-4-20250514", baseUrl: "" },
  { value: "gemini", label: "Google Gemini", provider: "gemini", model: "gemini-2.0-flash", baseUrl: "" },
  { value: "openrouter", label: "OpenRouter", provider: "openai-compatible", model: "openai/gpt-4.1-mini", baseUrl: "https://openrouter.ai/api/v1" },
  { value: "deepseek", label: "DeepSeek", provider: "openai-compatible", model: "deepseek-chat", baseUrl: "https://api.deepseek.com/v1" },
  { value: "moonshot", label: "Moonshot Kimi", provider: "openai-compatible", model: "moonshot-v1-8k", baseUrl: "https://api.moonshot.cn/v1" },
  { value: "qwen", label: "阿里 Qwen", provider: "openai-compatible", model: "qwen-plus", baseUrl: "https://dashscope.aliyuncs.com/compatible-mode/v1" },
  { value: "zhipu", label: "智谱 GLM", provider: "openai-compatible", model: "glm-4.5-flash", baseUrl: "https://open.bigmodel.cn/api/paas/v4" },
  { value: "minimax", label: "MiniMax", provider: "openai-compatible", model: "abab6.5-chat", baseUrl: "https://api.minimax.chat/v1" },
  { value: "ollama", label: "Ollama 本地", provider: "openai-compatible", model: "llama3.1", baseUrl: "http://localhost:11434/v1" },
  { value: "custom-compatible", label: "自定义 OpenAI-Compatible", provider: "openai-compatible", model: "", baseUrl: "" },
];

export function getPresetFromSettings(settings: AISettingsViewModel): AiPreset {
  if (settings.provider === "none") return "none";
  if (settings.provider === "openai") return "openai";
  if (settings.provider === "anthropic") return "anthropic";
  if (settings.provider === "gemini") return "gemini";
  const baseUrl = settings.baseUrl.trim().toLowerCase();
  if (baseUrl.includes("openrouter")) return "openrouter";
  if (baseUrl.includes("deepseek")) return "deepseek";
  if (baseUrl.includes("moonshot")) return "moonshot";
  if (baseUrl.includes("dashscope") || baseUrl.includes("qwen")) return "qwen";
  if (baseUrl.includes("bigmodel") || baseUrl.includes("zhipu")) return "zhipu";
  if (baseUrl.includes("minimax")) return "minimax";
  if (baseUrl.includes("11434") || baseUrl.includes("ollama")) return "ollama";
  return "custom-compatible";
}

export function applyPreset(preset: AiPreset, current: AISettingsViewModel): AISettingsViewModel {
  const match = aiProviderPresets.find((item) => item.value === preset) ?? aiProviderPresets[0];
  if (preset === "none") {
    return { provider: "none", model: "", apiKey: "", baseUrl: "" };
  }
  return {
    provider: match.provider,
    model: match.model || current.model,
    apiKey: current.apiKey,
    baseUrl: match.baseUrl,
  };
}

export function getModelOptions(preset: AiPreset): string[] {
  switch (preset) {
    case "openai":
      return ["gpt-4.1-mini", "gpt-4.1", "gpt-4o", "gpt-4o-mini"];
    case "anthropic":
      return ["claude-sonnet-4-20250514", "claude-haiku-4-20250514", "claude-opus-4-20250514"];
    case "gemini":
      return ["gemini-2.5-flash", "gemini-2.0-flash", "gemini-2.5-pro"];
    case "openrouter":
      return ["openai/gpt-4.1-mini", "openai/gpt-4o-mini", "anthropic/claude-sonnet-4", "google/gemini-2.0-flash"];
    case "deepseek":
      return ["deepseek-chat", "deepseek-reasoner"];
    case "moonshot":
      return ["moonshot-v1-8k", "moonshot-v1-32k"];
    case "qwen":
      return ["qwen-plus", "qwen-turbo", "qwen-max"];
    case "zhipu":
      return ["glm-4.5-flash", "glm-4.5", "glm-4-plus"];
    case "minimax":
      return ["abab6.5-chat", "abab6.5s-chat"];
    case "ollama":
      return ["llama3.1", "qwen2.5", "deepseek-r1"];
    default:
      return [];
  }
}
