import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { portugueseTranslation } from "../i18n/pt/translation";
import { App } from "./App";

describe("production application", () => {
  it("switches interface language without changing corpus content or workspace state", async () => {
    const user = userEvent.setup();
    window.history.replaceState({}, "", "/corpora/eu-data-protection");

    render(<App />);

    expect(
      await screen.findByRole("heading", {
        name: "European Data Protection Framework",
      }),
    ).toBeVisible();
    expect(document.documentElement).toHaveAttribute("lang", "en");

    await user.click(
      screen.getByRole("treeitem", {
        name: "PDF: General Data Protection Regulation",
      }),
    );
    await user.click(screen.getByRole("tab", { name: "Chat" }));
    const composer = screen.getByRole("textbox", { name: "Research question" });
    await user.type(composer, "Preserved research draft");

    await user.selectOptions(
      screen.getByRole("combobox", { name: "Interface language" }),
      "pt",
    );

    expect(screen.getByRole("link", { name: "Todos os corpus" })).toBeVisible();
    expect(document.documentElement).toHaveAttribute("lang", "pt");
    expect(
      screen.getByRole("heading", {
        name: "European Data Protection Framework",
      }),
    ).toBeVisible();
    expect(screen.getByText("Regulation (EU) 2016/679")).toBeInTheDocument();
    expect(
      screen.getByRole("textbox", {
        name: portugueseTranslation.chat.composerLabel,
      }),
    ).toHaveValue("Preserved research draft");
    expect(
      screen.getByText(portugueseTranslation.app.notLegalAdvice),
    ).toBeVisible();
  });
});
