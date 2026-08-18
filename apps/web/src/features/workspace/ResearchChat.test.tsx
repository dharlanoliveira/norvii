import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  expectNoAccessibilityViolations,
  renderAtRoute,
} from "../../test/render";
import { ResearchChat } from "./ResearchChat";

describe("research chat availability", () => {
  it("honestly communicates that grounded chat is not part of this feature", async () => {
    const result = renderAtRoute(<ResearchChat />);

    expect(
      screen.getByRole("heading", { name: "Grounded chat is coming next." }),
    ).toBeVisible();
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
    expect(screen.queryByText(/simulated/i)).not.toBeInTheDocument();
    await expectNoAccessibilityViolations(result.container);
  });
});
