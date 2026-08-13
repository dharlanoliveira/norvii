import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { Route, Routes, useNavigate } from "react-router-dom";

import { AppRoutes } from "../../app/routes";
import { createDemonstrationCatalog } from "../../research/demonstration/createDemonstrationCatalog";
import {
  expectNoAccessibilityViolations,
  renderAtRoute,
} from "../../test/render";
import { CorpusWorkspacePage } from "./CorpusWorkspacePage";

describe("corpus research workspace", () => {
  it("starts with isolated state when direct navigation changes the corpus", async () => {
    const user = userEvent.setup();
    const brazilCorpus = createDemonstrationCatalog().findCorpus(
      "brazil-data-protection",
    );
    if (!brazilCorpus) throw new Error("Expected demonstration corpus.");

    function DirectCorpusNavigation() {
      const navigate = useNavigate();
      return (
        <button
          type="button"
          onClick={() => void navigate("/corpora/brazil-data-protection")}
        >
          Change active corpus
        </button>
      );
    }

    renderAtRoute(
      <>
        <DirectCorpusNavigation />
        <AppRoutes />
      </>,
      "/corpora/eu-data-protection",
    );

    await user.click(
      await screen.findByRole("treeitem", {
        name: "PDF: General Data Protection Regulation",
      }),
    );
    expect(screen.getByRole("tab", { name: "Source" })).toHaveAttribute(
      "aria-selected",
      "true",
    );

    await user.click(
      screen.getByRole("button", { name: "Change active corpus" }),
    );

    expect(
      await screen.findByRole("heading", {
        name: brazilCorpus.name,
      }),
    ).toBeVisible();
    expect(screen.getByRole("tab", { name: "Chat" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
  });

  it("preserves draft and conversation while citations open source locations", async () => {
    const user = userEvent.setup();
    const result = renderAtRoute(<AppRoutes />, "/corpora/eu-data-protection");

    await user.click(
      await screen.findByRole("treeitem", {
        name: "PDF: General Data Protection Regulation",
      }),
    );
    expect(
      screen.getByRole("heading", {
        name: "General Data Protection Regulation",
      }),
    ).toBeVisible();

    await user.click(screen.getByRole("tab", { name: "Chat" }));
    const composer = screen.getByRole("textbox", { name: "Research question" });
    await user.type(composer, "Draft question");
    await user.click(screen.getByRole("tab", { name: "Source" }));
    await user.click(screen.getByRole("tab", { name: "Chat" }));
    expect(composer).toHaveValue("Draft question");

    await user.clear(composer);
    await user.type(
      composer,
      "What principles govern personal data processing?",
    );
    await user.click(screen.getByRole("button", { name: "Send question" }));
    expect(
      await screen.findByText(/Article 5 establishes lawfulness/),
    ).toBeVisible();

    fireEvent.click(
      screen.getByRole("button", {
        name: "Open citation GDPR, Article 5",
      }),
    );

    expect(screen.getByRole("tab", { name: "Source" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(screen.getAllByText("Article 5")).toHaveLength(2);

    await user.click(screen.getByRole("tab", { name: "Chat" }));
    expect(screen.getByText(/Article 5 establishes lawfulness/)).toBeVisible();

    await expectNoAccessibilityViolations(result.container);
  });

  it("reports an unavailable citation without discarding the conversation", async () => {
    const user = userEvent.setup();
    const catalog = createDemonstrationCatalog();
    const resolveCitation = vi.fn(() => undefined);
    const unresolvedCatalog = { ...catalog, resolveCitation };

    renderAtRoute(
      <Routes>
        <Route
          path="corpora/:corpusId"
          element={<CorpusWorkspacePage catalog={unresolvedCatalog} />}
        />
      </Routes>,
      "/corpora/eu-data-protection",
    );

    const composer = screen.getByRole("textbox", { name: "Research question" });
    await user.type(
      composer,
      "What principles govern personal data processing?",
    );
    await user.click(screen.getByRole("button", { name: "Send question" }));
    fireEvent.click(
      await screen.findByRole("button", {
        name: "Open citation GDPR, Article 5",
      }),
    );

    expect(resolveCitation).toHaveBeenCalledOnce();
    expect(
      screen.getByRole("alert", {
        name: "The cited location is unavailable.",
      }),
    ).toBeVisible();
    expect(screen.getByText(/Article 5 establishes lawfulness/)).toBeVisible();
    expect(screen.getByRole("tab", { name: "Chat" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
  });
});
