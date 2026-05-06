import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import App from "./App";

describe("App routes", () => {
  it("renders the primary navigation, route surfaces, and mocked status content", async () => {
    const user = userEvent.setup();

    render(<App />);

    expect(await screen.findByRole("link", { name: "概览" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "代理" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "技能" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "商店" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "项目" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "助手" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "设置" })).toBeInTheDocument();

    expect(screen.getByText("查看本机 AI 代理、已安装技能和任务执行状态。")).toBeInTheDocument();

    await user.click(screen.getByRole("link", { name: "代理" }));
    expect(await screen.findByText("管理本机已安装的 AI 代理，查看运行状态和技能目录。")).toBeInTheDocument();

    await user.click(screen.getByRole("link", { name: "技能" }));
    expect(await screen.findByText("暂无已安装的技能")).toBeInTheDocument();

    await user.click(screen.getByRole("link", { name: "商店" }));
    expect(await screen.findByText("浏览与安装技能")).toBeInTheDocument();

    await user.click(screen.getByRole("link", { name: "项目" }));
    expect(await screen.findByText("暂无项目")).toBeInTheDocument();

    await user.click(screen.getByRole("link", { name: "助手" }));
    expect(await screen.findByText("任务执行助手")).toBeInTheDocument();

    await user.click(screen.getByRole("link", { name: "设置" }));
    expect(await screen.findByText("系统诊断")).toBeInTheDocument();
  });
});
