import axe from "axe-core";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { i18n } from "../i18n/config";
import { App } from "./App";

describe("App", () => {
  it("starts in English and changes all product navigation to Portuguese", async () => {
    const user = userEvent.setup();
    render(<App />);

    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent(
      "Choose the body of law",
    );
    await user.selectOptions(
      screen.getByRole("combobox", { name: /interface language/i }),
      "pt",
    );

    expect(
      await screen.findAllByRole("link", { name: "Abrir corpus" }),
    ).toHaveLength(2);
    expect(i18n.resolvedLanguage).toBe("pt");
  });

  it("has no automatically detectable accessibility violations on the catalog", async () => {
    const { container } = render(<App />);
    const results = await axe.run(container, {
      rules: { "color-contrast": { enabled: false } },
    });
    expect(results.violations).toEqual([]);
  });
});
