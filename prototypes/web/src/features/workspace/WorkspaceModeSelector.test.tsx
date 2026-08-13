import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { renderAtRoute } from "../../test/render";
import { WorkspaceModeSelector } from "./WorkspaceModeSelector";

describe("WorkspaceModeSelector", () => {
  it("announces and changes the active research mode", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderAtRoute(<WorkspaceModeSelector mode="chat" onChange={onChange} />);

    expect(screen.getByRole("tab", { name: "Chat" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    await user.click(screen.getByRole("tab", { name: "Source" }));
    expect(onChange).toHaveBeenCalledWith("source");
  });
});
