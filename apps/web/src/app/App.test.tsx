import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { portugueseTranslation } from "../i18n/pt/translation";
import { App } from "./App";

describe("production application", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("shows authoritative failure without runtime fixture fallback", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>().mockRejectedValue(new Error("offline")),
    );
    window.history.replaceState({}, "", "/");

    render(<App />);

    expect(await screen.findByRole("alert")).toBeVisible();
    expect(
      screen.queryByText("European Data Protection Framework"),
    ).not.toBeInTheDocument();
    expect(screen.queryAllByRole("article")).toHaveLength(0);
  });

  it("switches interface language while authoritative services are unavailable", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>().mockRejectedValue(new Error("offline")),
    );
    const user = userEvent.setup();
    window.history.replaceState({}, "", "/");
    render(<App />);

    await screen.findByRole("alert");
    await user.selectOptions(
      screen.getByRole("combobox", { name: "Interface language" }),
      "pt",
    );

    expect(document.documentElement).toHaveAttribute("lang", "pt");
    expect(
      screen.getByText(portugueseTranslation.app.notLegalAdvice),
    ).toBeVisible();
    expect(screen.getByRole("alert")).toHaveTextContent(
      portugueseTranslation.catalog.loadFailed,
    );
  });
});
