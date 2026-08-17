import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes } from "react-router-dom";
import { describe, expect, it } from "vitest";

import { renderAtRoute } from "../../test/render";
import { CorpusCatalogPage } from "./CorpusCatalogPage";

describe("CorpusCatalogPage", () => {
  it("presents two language-labelled corpora and opens the selected route", async () => {
    const user = userEvent.setup();
    renderAtRoute(
      <Routes>
        <Route index element={<CorpusCatalogPage />} />
        <Route path="corpora/:corpusId" element={<p>Workspace opened</p>} />
      </Routes>,
    );

    expect(screen.getAllByRole("article")).toHaveLength(2);
    expect(screen.getByText("Portuguese")).toBeInTheDocument();
    expect(screen.getByText("English")).toBeInTheDocument();

    const [firstCorpusLink] = screen.getAllByRole("link", {
      name: "Open corpus",
    });
    if (!firstCorpusLink) {
      throw new Error("Expected the first corpus link to be present.");
    }

    await user.click(firstCorpusLink);
    expect(screen.getByText("Workspace opened")).toBeInTheDocument();
  });
});
