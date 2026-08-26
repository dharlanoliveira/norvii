import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes } from "react-router-dom";
import { describe, expect, it } from "vitest";

import { corpora } from "../../fixtures/legal-content/corpora";
import { renderAtRoute } from "../../test/render";
import { CorpusWorkspacePage } from "../workspace/CorpusWorkspacePage";
import { CorpusCatalogPage } from "./CorpusCatalogPage";

describe("CorpusCatalogPage", () => {
  it("presents every catalog corpus and opens the selected route", async () => {
    const user = userEvent.setup();
    renderAtRoute(
      <Routes>
        <Route index element={<CorpusCatalogPage />} />
        <Route path="corpora/:corpusId" element={<p>Workspace opened</p>} />
      </Routes>,
    );

    expect(screen.getAllByRole("article")).toHaveLength(corpora.length);
    expect(screen.getAllByText("Portuguese")).toHaveLength(2);
    expect(screen.getAllByText("English")).toHaveLength(2);

    const [firstCorpusLink] = screen.getAllByRole("link", {
      name: "Open corpus",
    });
    if (!firstCorpusLink) {
      throw new Error("Expected the first corpus link to be present.");
    }

    await user.click(firstCorpusLink);
    expect(screen.getByText("Workspace opened")).toBeInTheDocument();
  });

  it.each([
    "brazil-anti-corruption-white-collar-crime",
    "us-fair-housing-disability-accommodations",
  ])("opens %s through its catalog link", async (corpusId) => {
    const user = userEvent.setup();
    const selectedCorpus = corpora.find((corpus) => corpus.id === corpusId);

    if (!selectedCorpus) {
      throw new Error(`Expected catalog corpus ${corpusId}.`);
    }

    renderAtRoute(
      <Routes>
        <Route index element={<CorpusCatalogPage />} />
        <Route path="corpora/:corpusId" element={<CorpusWorkspacePage />} />
      </Routes>,
    );

    const matchingLink = screen
      .getAllByRole("link", { name: "Open corpus" })
      .find((link) => link.getAttribute("href") === `/corpora/${corpusId}`);

    if (!matchingLink) {
      throw new Error(`Expected a catalog link for ${corpusId}.`);
    }

    await user.click(matchingLink);
    expect(
      await screen.findByRole("heading", { name: selectedCorpus.label }),
    ).toBeInTheDocument();
  });
});
