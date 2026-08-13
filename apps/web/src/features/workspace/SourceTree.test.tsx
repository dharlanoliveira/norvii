import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { createDemonstrationCatalog } from "../../research/demonstration/createDemonstrationCatalog";
import {
  expectNoAccessibilityViolations,
  renderAtRoute,
} from "../../test/render";
import { SourceTree } from "./SourceTree";

describe("source tree", () => {
  it("uses a single tab stop and the complete ARIA tree navigation contract", async () => {
    const user = userEvent.setup();
    const corpus =
      createDemonstrationCatalog().findCorpus("eu-data-protection");
    if (!corpus) throw new Error("Expected demonstration corpus.");

    renderAtRoute(
      <SourceTree
        corpus={corpus}
        selectedSourceId={undefined}
        onSelect={vi.fn()}
      />,
    );

    const root = screen.getByRole("treeitem", { name: corpus.name });
    const pdfGroup = screen.getByRole("treeitem", {
      name: "PDF documents, 1 source",
    });
    const pdf = screen.getByRole("treeitem", {
      name: "PDF: General Data Protection Regulation",
    });
    const lastSource = screen.getByRole("treeitem", {
      name: "Link: Guidelines on personal data breach notification",
    });

    expect(root).toHaveAttribute("tabindex", "0");
    expect(pdfGroup).toHaveAttribute("tabindex", "-1");

    root.focus();
    await user.keyboard("{ArrowDown}");
    expect(pdfGroup).toHaveFocus();
    await user.keyboard("{ArrowRight}");
    expect(pdf).toHaveFocus();
    await user.keyboard("{ArrowLeft}{ArrowLeft}");
    expect(pdfGroup).toHaveFocus();
    expect(pdfGroup).toHaveAttribute("aria-expanded", "false");
    await user.keyboard("{ArrowRight}{End}");
    expect(lastSource).toHaveFocus();
    await user.keyboard("{Home}");
    expect(root).toHaveFocus();
  });

  it("supports hierarchical keyboard selection and group collapse", async () => {
    const user = userEvent.setup();
    const corpus =
      createDemonstrationCatalog().findCorpus("eu-data-protection");
    const onSelect = vi.fn();

    if (!corpus) throw new Error("Expected demonstration corpus.");

    const result = renderAtRoute(
      <SourceTree
        corpus={corpus}
        selectedSourceId={undefined}
        onSelect={onSelect}
      />,
    );

    const pdfGroup = screen.getByRole("treeitem", {
      name: "PDF documents, 1 source",
    });
    pdfGroup.focus();
    await user.keyboard("{ArrowDown}{Enter}");

    expect(onSelect).toHaveBeenCalledWith("gdpr");

    const externalGroup = screen.getByRole("treeitem", {
      name: "External links, 2 sources",
    });

    expect(
      screen.getByRole("treeitem", {
        name: "Link: Guidelines on controller and processor concepts",
        description: "Available",
      }),
    ).toBeVisible();
    expect(
      screen.getByRole("treeitem", {
        name: "Link: Guidelines on personal data breach notification",
        description: "Preview unavailable",
      }),
    ).toBeVisible();

    await user.click(externalGroup);

    expect(
      screen.queryByRole("treeitem", {
        name: /Guidelines on controller and processor concepts/,
      }),
    ).not.toBeInTheDocument();

    await expectNoAccessibilityViolations(result.container);
  });
});
