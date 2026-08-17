import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { corpora } from "../../fixtures/legal-content/corpora";
import { renderAtRoute } from "../../test/render";
import { SourceViewer } from "./SourceViewer";

describe("SourceViewer", () => {
  it("renders a selected PDF at its stable section", () => {
    const source = corpora.at(0)?.sources.at(0);
    if (!source) {
      throw new Error("Expected the LGPD source fixture to be present.");
    }
    renderAtRoute(
      <SourceViewer
        source={source}
        sectionId="article-18"
        onSelectSection={vi.fn()}
      />,
    );

    expect(
      screen.getByRole("heading", { name: "Direitos do titular" }),
    ).toBeInTheDocument();
    expect(screen.getByText(/Current location/)).toHaveTextContent("Art. 18");
  });

  it("renders a recoverable unavailable external preview", () => {
    const source = corpora.at(0)?.sources.at(2);
    if (!source) {
      throw new Error("Expected the unavailable source fixture to be present.");
    }
    renderAtRoute(
      <SourceViewer
        source={source}
        sectionId={null}
        onSelectSection={vi.fn()}
      />,
    );

    expect(
      screen.getByRole("heading", { name: /preview is unavailable/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /open original source/i }),
    ).toHaveAttribute("target", "_blank");
  });
});
