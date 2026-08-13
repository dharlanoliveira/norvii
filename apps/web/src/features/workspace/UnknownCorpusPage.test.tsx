import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  expectNoAccessibilityViolations,
  renderAtRoute,
} from "../../test/render";
import { UnknownCorpusPage } from "./UnknownCorpusPage";

describe("unknown corpus recovery", () => {
  it("explains the unresolved route and returns to the catalog", async () => {
    const result = renderAtRoute(<UnknownCorpusPage />, "/corpora/not-found");

    expect(
      screen.getByRole("heading", { name: "This corpus could not be found." }),
    ).toBeVisible();
    expect(
      screen.getByRole("link", { name: "Return to corpus catalog" }),
    ).toHaveAttribute("href", "/");

    await expectNoAccessibilityViolations(result.container);
  });
});
