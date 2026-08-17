import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { createDemonstrationCatalog } from "../../research/demonstration/createDemonstrationCatalog";
import {
  expectNoAccessibilityViolations,
  renderAtRoute,
} from "../../test/render";
import { SourceViewer } from "./SourceViewer";

describe("source viewer", () => {
  it("presents PDF evidence and moves through stable locations", async () => {
    const user = userEvent.setup();
    const corpus =
      createDemonstrationCatalog().findCorpus("eu-data-protection");
    const source = corpus?.sources.find((candidate) => candidate.id === "gdpr");
    const onLocationChange = vi.fn();

    if (!source) throw new Error("Expected demonstration PDF.");

    const result = renderAtRoute(
      <SourceViewer
        source={source}
        activeLocationId="article-5"
        onLocationChange={onLocationChange}
      />,
    );

    expect(
      screen.getByRole("heading", {
        name: "General Data Protection Regulation",
      }),
    ).toBeVisible();
    expect(screen.getAllByText("Article 5")).toHaveLength(2);
    expect(
      screen.getByText(/Personal data shall be processed lawfully/),
    ).toBeVisible();

    await user.click(screen.getByRole("button", { name: "Next location" }));

    expect(onLocationChange).toHaveBeenCalledWith("article-24");
    await expectNoAccessibilityViolations(result.container);
  });

  it("presents safe available and unavailable external source states", async () => {
    const corpus =
      createDemonstrationCatalog().findCorpus("eu-data-protection");
    const available = corpus?.sources.find(
      (candidate) => candidate.id === "edpb-controller-guidelines",
    );
    const unavailable = corpus?.sources.find(
      (candidate) => candidate.id === "edpb-breach-guidelines",
    );

    if (!available || !unavailable) {
      throw new Error("Expected demonstration external sources.");
    }

    const result = renderAtRoute(
      <SourceViewer
        source={available}
        activeLocationId={undefined}
        onLocationChange={vi.fn()}
      />,
    );

    expect(screen.getByText("Captured overview")).toBeVisible();
    const officialLink = screen.getByRole("link", {
      name: "Open official source",
    });
    expect(officialLink).toHaveAttribute("target", "_blank");
    expect(officialLink).toHaveAttribute("rel", "noopener noreferrer");

    result.rerender(
      <SourceViewer
        source={unavailable}
        activeLocationId={undefined}
        onLocationChange={vi.fn()}
      />,
    );

    expect(screen.getByText("Preview unavailable")).toBeVisible();
    expect(screen.getByText(/does not embed the remote page/)).toBeVisible();
    await expectNoAccessibilityViolations(result.container);
  });
});
