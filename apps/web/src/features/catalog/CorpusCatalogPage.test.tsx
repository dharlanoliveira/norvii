import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { createDemonstrationCatalog } from "../../research/demonstration/createDemonstrationCatalog";
import {
  expectNoAccessibilityViolations,
  renderAtRoute,
} from "../../test/render";
import { CorpusCatalogPage } from "./CorpusCatalogPage";

describe("corpus catalog", () => {
  it("presents the two isolated legal collections and their entry actions", async () => {
    const catalog = createDemonstrationCatalog();
    const portugueseCorpus = catalog
      .listCorpora()
      .find((corpus) => corpus.language === "pt");
    if (!portugueseCorpus) throw new Error("Expected Portuguese corpus.");
    const result = renderAtRoute(<CorpusCatalogPage catalog={catalog} />);

    expect(
      screen.getByRole("heading", {
        name: "Choose the boundary for your research.",
      }),
    ).toBeVisible();
    expect(screen.getAllByRole("article")).toHaveLength(2);
    expect(
      screen.getByRole("heading", {
        name: portugueseCorpus.name,
      }),
    ).toBeVisible();
    expect(
      screen.getByRole("heading", {
        name: "European Data Protection Framework",
      }),
    ).toBeVisible();
    expect(screen.getByText("Portuguese")).toBeVisible();
    expect(screen.getByText("English")).toBeVisible();
    expect(screen.getByText("Brazil")).toBeVisible();
    expect(screen.getByText("European Union")).toBeVisible();
    expect(screen.getByText("2 sources")).toBeVisible();
    expect(screen.getByText("3 sources")).toBeVisible();
    expect(
      screen.getByRole("link", {
        name: `Open corpus ${portugueseCorpus.name}`,
      }),
    ).toHaveAttribute("href", "/corpora/brazil-data-protection");

    await expectNoAccessibilityViolations(result.container);
  });
});
