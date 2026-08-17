import {
  fireEvent,
  screen,
  waitForElementToBeRemoved,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { createDemonstrationCatalog } from "../../research/demonstration/createDemonstrationCatalog";
import {
  expectNoAccessibilityViolations,
  renderAtRoute,
} from "../../test/render";
import { ResearchChat } from "./ResearchChat";

describe("research chat", () => {
  it("renders a structured simulated answer and opens its citation", async () => {
    const user = userEvent.setup();
    const corpus =
      createDemonstrationCatalog().findCorpus("eu-data-protection");
    const onOpenCitation = vi.fn();
    if (!corpus) throw new Error("Expected demonstration corpus.");

    const { container } = renderAtRoute(
      <ResearchChat corpus={corpus} onOpenCitation={onOpenCitation} />,
    );

    await expectNoAccessibilityViolations(container);

    await user.type(
      screen.getByRole("textbox", { name: "Research question" }),
      "What principles govern personal data processing?",
    );
    await user.click(screen.getByRole("button", { name: "Send question" }));

    expect(
      await screen.findByText(/Article 5 establishes lawfulness/),
    ).toBeVisible();
    expect(screen.getByText("Simulated response")).toBeVisible();

    fireEvent.click(
      screen.getByRole("button", {
        name: "Open citation GDPR, Article 5",
      }),
    );

    expect(onOpenCitation).toHaveBeenCalledWith({
      id: "gdpr-article-5",
      sourceId: "gdpr",
      locationId: "article-5",
      label: "GDPR, Article 5",
    });
  });

  it("abstains without evidence and keeps a failed question recoverable", async () => {
    const user = userEvent.setup();
    const corpus =
      createDemonstrationCatalog().findCorpus("eu-data-protection");
    if (!corpus) throw new Error("Expected demonstration corpus.");

    renderAtRoute(<ResearchChat corpus={corpus} onOpenCitation={vi.fn()} />);
    const composer = screen.getByRole("textbox", { name: "Research question" });

    await user.type(composer, "Who won the latest election?");
    await user.click(screen.getByRole("button", { name: "Send question" }));

    expect(
      await screen.findByText(/Norvii abstains rather than infer/),
    ).toBeVisible();

    await user.type(composer, "Please simulate failure");
    await user.click(screen.getByRole("button", { name: "Send question" }));

    expect(
      await screen.findByText("The simulated response could not be completed."),
    ).toBeVisible();
    expect(screen.getByText("Please simulate failure")).toBeVisible();

    await user.click(screen.getByRole("button", { name: "Retry question" }));

    await waitForElementToBeRemoved(await screen.findByRole("status"));

    expect(await screen.findByText(/The retry completed/)).toBeVisible();
    expect(
      screen.queryByText("The simulated response could not be completed."),
    ).not.toBeInTheDocument();
    expect(screen.getAllByText("Please simulate failure")).toHaveLength(1);
  });
});
