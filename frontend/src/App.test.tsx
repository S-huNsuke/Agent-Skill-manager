import { render, screen } from "@testing-library/react";
import { vi } from "vitest";
import App from "./App";

vi.mock("./lib/api", async () => {
  const actual = await vi.importActual<typeof import("./lib/api")>("./lib/api");
  return {
    ...actual,
    waitForApi: vi.fn(async () => actual.mockApi),
  };
});

describe("App routes", () => {
  it("renders the primary navigation and main application structure", async () => {
    render(<App />);

    // 验证所有主导航链接存在
    expect(await screen.findByRole("link", { name: "概览" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "代理" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "技能" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "商店" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "项目" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "设置" })).toBeInTheDocument();

    // 验证应用标题存在
    expect(screen.getByText("技能管理器")).toBeInTheDocument();

    // 验证 AI 助手面板存在（使用 getAllByText 因为有多个相同文本）
    expect(screen.getAllByText("告诉我你想做什么").length).toBeGreaterThan(0);
  });
});
