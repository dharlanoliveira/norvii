import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderAtRoute } from "../../test/render";
import { UnknownCorpusPage } from "./UnknownCorpusPage";

describe("UnknownCorpusPage", () => {
  it("offers a clear route back to the catalog", () => {
    renderAtRoute(<UnknownCorpusPage />, "/corpora/missing");

    expect(
      screen.getByRole("heading", { name: /not available/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /return to corpus catalog/i }),
    ).toHaveAttribute("href", "/");
  });
});
