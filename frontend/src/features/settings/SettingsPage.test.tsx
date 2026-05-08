import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { vi } from "vitest";
import { SettingsPage } from "./SettingsPage";

vi.mock("../../lib/api", async () => {
  const actual = await vi.importActual<typeof import("../../lib/api")>("../../lib/api");
  return {
    ...actual,
    waitForApi: vi.fn(async () => ({
      getGeneralSettings: async () => {
        throw new Error("general settings unavailable");
      },
      getAutomationSettings: async () => {
        throw new Error("automation settings unavailable");
      },
      getAISettings: async () => {
        throw new Error("ai settings unavailable");
      },
      getAppInfoFull: async () => {
        throw new Error("app info unavailable");
      },
    })),
  };
});

describe("SettingsPage", () => {
  it("keeps AI controls visible even if settings loading fails", async () => {
    const user = userEvent.setup();

    render(<SettingsPage diagnostics={[]} />);

    await user.click(await screen.findByRole("button", { name: "AI 配置" }));

    await waitFor(() => {
      expect(screen.getByLabelText("供应商预设")).toBeInTheDocument();
      expect(screen.getByLabelText("API Key")).toBeInTheDocument();
      expect(screen.getByLabelText("模型预设")).toBeInTheDocument();
    });
  });
});
