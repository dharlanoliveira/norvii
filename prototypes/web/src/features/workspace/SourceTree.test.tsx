import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { corpora } from "../../fixtures/legal-content/corpora";
import { renderAtRoute } from "../../test/render";
import { SourceTree } from "./SourceTree";

describe("SourceTree", () => {
  it("expands groups and selects a source using the keyboard", async () => {
    const user = userEvent.setup();
    const onSelectSource = vi.fn();
    const corpus = corpora.at(0);
    if (!corpus) {
      throw new Error("Expected the Brazilian corpus fixture to be present.");
    }
    renderAtRoute(
      <SourceTree
        corpus={corpus}
        selectedSourceId={null}
        onSelectSource={onSelectSource}
      />,
    );

    const pdfGroup = screen.getByRole("treeitem", { name: /pdf documents/i });
    expect(pdfGroup).toHaveAttribute("aria-expanded", "true");
    await user.click(pdfGroup);
    expect(pdfGroup).toHaveAttribute("aria-expanded", "false");
    await user.click(pdfGroup);

    const source = screen.getByRole("treeitem", { name: /lgpd.*available/i });
    source.focus();
    await user.keyboard("{Enter}");
    expect(onSelectSource).toHaveBeenCalledWith("lgpd-law");
  });
});
