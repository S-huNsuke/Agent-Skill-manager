import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { vi } from "vitest";
import { AssistantPanel } from "./AssistantPanel";
import type { AISettingsViewModel, AssistantTaskViewModel } from "../../lib/types";

const mockSubmitGoal = vi.hoisted(() => vi.fn());
const mockChatAssistant = vi.hoisted(() => vi.fn());
const mockSaveAISettings = vi.hoisted(() => vi.fn());
const mockGetAISettings = vi.hoisted(() => vi.fn());
const mockGetTaskHistory = vi.hoisted(() => vi.fn());
const mockDeleteTaskHistoryItem = vi.hoisted(() => vi.fn());
const mockGetSuggestionTemplates = vi.hoisted(() => vi.fn());

vi.mock("../../lib/api", async () => {
  const actual = await vi.importActual<typeof import("../../lib/api")>("../../lib/api");
  return {
      ...actual,
      waitForApi: vi.fn(async () => ({
        getSuggestionTemplates: mockGetSuggestionTemplates,
      getTaskHistory: mockGetTaskHistory,
      deleteTaskHistoryItem: mockDeleteTaskHistoryItem,
      getAISettings: mockGetAISettings,
      saveAISettings: mockSaveAISettings,
      submitGoal: mockSubmitGoal,
      chatAssistant: mockChatAssistant,
    })),
  };
});

describe("AssistantPanel", () => {
  const idleTask: AssistantTaskViewModel = {
    id: "assistant-idle",
    request: "",
    status: "queued",
    nextStep: "等待用户输入目标",
    summary: "AI 助手待命中，输入目标即可开始规划。",
    recommendation: "",
    reason: "",
    records: [],
    planJson: "",
    planSteps: [],
    resolvedActions: [],
  };

  beforeEach(() => {
    vi.clearAllMocks();
    try {
      localStorage.removeItem("agent-skills-manager.chatSessions");
    } catch {
      // 测试环境中 localStorage 可能不可用
    }
    const defaultSettings: AISettingsViewModel = { provider: "none", model: "", apiKey: "", baseUrl: "" };
    mockGetAISettings.mockResolvedValue(defaultSettings);
    mockGetTaskHistory.mockResolvedValue([]);
    mockGetSuggestionTemplates.mockResolvedValue([]);
    mockDeleteTaskHistoryItem.mockResolvedValue("ok");
    mockSaveAISettings.mockResolvedValue("ok");
    mockChatAssistant.mockImplementation(async (message: string) => ({
      reply: `聊天回复：${message}`,
      provider: "none",
      model: "",
    }));
    mockSubmitGoal.mockImplementation(async (goal: string) => ({
      id: "assistant-active",
      request: goal,
      status: "planning",
      nextStep: "等待用户确认",
      summary: "这是规划摘要",
      recommendation: "继续执行计划",
      reason: "",
      records: [],
      planJson: "",
      planSteps: [],
      resolvedActions: [],
    }));
  });

  it("creates a planning task when the user chooses the planning action", async () => {
    const user = userEvent.setup();

    render(<AssistantPanel task={idleTask} />);

    await user.type(screen.getByPlaceholderText("输入消息..."), "安装一组技能");
    await user.click(screen.getByRole("button", { name: "规划技能方案" }));

    expect(mockSubmitGoal).toHaveBeenCalledWith("安装一组技能");
    expect(await screen.findByText("这是规划摘要")).toBeInTheDocument();
  });

  it("keeps provider configuration collapsed until the user opens it", async () => {
    const user = userEvent.setup();

    render(<AssistantPanel task={idleTask} />);

    expect(screen.getByRole("button", { name: "展开 AI 配置" })).toBeInTheDocument();
    expect(screen.queryByLabelText("供应商预设")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "展开 AI 配置" }));
    await user.selectOptions(screen.getByLabelText("供应商预设"), "openai");
    await user.clear(screen.getByLabelText("模型"));
    await user.type(screen.getByLabelText("模型"), "gpt-4.1-mini");
    await user.type(screen.getByLabelText("API Key"), "sk-test");
    await user.click(screen.getByRole("button", { name: "保存 AI 配置" }));

    expect(mockSaveAISettings).toHaveBeenCalledWith({
      provider: "openai",
      model: "gpt-4.1-mini",
      apiKey: "sk-test",
      baseUrl: "",
    });
    expect(await screen.findByText("AI 配置已保存")).toBeInTheDocument();
  });

  it("sends regular messages through the chat API by default", async () => {
    const user = userEvent.setup();

    render(<AssistantPanel task={idleTask} />);

    await user.type(screen.getByPlaceholderText("输入消息..."), "你好");
    await user.click(screen.getByTitle("发送"));

    expect(mockChatAssistant).toHaveBeenCalledWith("你好", []);
    expect(mockSubmitGoal).not.toHaveBeenCalled();
    expect(screen.queryByText("正在思考...")).not.toBeInTheDocument();
    expect(await screen.findByText("聊天回复：你好")).toBeInTheDocument();
  });

  it("starts planning directly from suggestion templates", async () => {
    const user = userEvent.setup();
    mockGetSuggestionTemplates.mockResolvedValue([
      {
        id: "tpl-test",
        category: "测试",
        title: "添加测试技能",
        description: "安装测试相关技能",
        promptTemplate: "为项目添加测试技能",
        isBuiltin: true,
      },
    ]);

    render(<AssistantPanel task={idleTask} />);

    await user.click(await screen.findByText("添加测试技能"));

    expect(mockSubmitGoal).toHaveBeenCalledWith("为项目添加测试技能");
    expect(mockChatAssistant).not.toHaveBeenCalled();
    expect(await screen.findByText("这是规划摘要")).toBeInTheDocument();
  });

  it("selects and deletes conversations from the history list", async () => {
    const user = userEvent.setup();

    render(<AssistantPanel task={idleTask} />);

    expect(screen.queryByRole("button", { name: "删除历史对话" })).not.toBeInTheDocument();
    await user.type(screen.getByPlaceholderText("输入消息..."), "你好");
    await user.click(screen.getByTitle("发送"));
    expect(await screen.findByText("聊天回复：你好")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "打开对话：你好" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "新对话" }));
    await user.type(screen.getByPlaceholderText("输入消息..."), "第二个问题");
    await user.click(screen.getByTitle("发送"));
    const messagesRegion = screen.getByRole("log", { name: "当前对话" });
    expect(await within(messagesRegion).findByText("第二个问题")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "打开对话：你好" }));
    expect(within(messagesRegion).getByText("你好")).toBeInTheDocument();
    expect(within(messagesRegion).queryByText("第二个问题")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "删除对话：你好" }));

    expect(screen.queryByText("你好")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "打开对话：你好" })).not.toBeInTheDocument();
    expect(screen.getAllByText("第二个问题").length).toBeGreaterThan(0);
  });

  it("deletes task history items from the recent tasks list", async () => {
    const user = userEvent.setup();
    mockGetTaskHistory.mockResolvedValue([
      {
        id: "task-1",
        goal: "安装技能任务",
        status: "completed",
        startedAt: "2026-05-08 10:00",
        finishedAt: "2026-05-08 10:01",
        summary: "完成",
      },
    ]);

    render(<AssistantPanel task={idleTask} />);

    expect(await screen.findByText("安装技能任务")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "删除任务：安装技能任务" }));

    expect(mockDeleteTaskHistoryItem).toHaveBeenCalledWith("task-1");
    expect(screen.queryByText("安装技能任务")).not.toBeInTheDocument();
  });
});
