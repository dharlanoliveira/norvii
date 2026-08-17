import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { corpora } from "../../fixtures/legal-content/corpora";
import { renderAtRoute } from "../../test/render";
import { ResearchChat } from "./ResearchChat";

describe("ResearchChat", () => {
  it("returns a deterministic citation for a prepared question", async () => {
    const user = userEvent.setup();
    const corpus = corpora.at(0);
    if (!corpus) {
      throw new Error("Expected the Brazilian corpus fixture to be present.");
    }
    renderAtRoute(<ResearchChat corpus={corpus} onOpenCitation={vi.fn()} />);

    await user.type(
      screen.getByRole("textbox", { name: /question for this corpus/i }),
      "What rights does a data subject have?",
    );
    await user.click(screen.getByRole("button", { name: /send question/i }));

    expect(await screen.findByRole("status")).toHaveTextContent(
      /reviewing the prepared evidence/i,
    );
    expect(
      await screen.findByRole("button", {
        name: /open citation lgpd, art. 18/i,
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/confirmation that processing exists/i),
    ).toBeInTheDocument();
  });

  it("abstains when the prepared corpus cannot support a question", async () => {
    const user = userEvent.setup();
    const corpus = corpora.at(0);
    if (!corpus) {
      throw new Error("Expected the Brazilian corpus fixture to be present.");
    }
    renderAtRoute(<ResearchChat corpus={corpus} onOpenCitation={vi.fn()} />);

    await user.type(
      screen.getByRole("textbox", { name: /question for this corpus/i }),
      "Tax law?",
    );
    await user.click(screen.getByRole("button", { name: /send question/i }));

    expect(
      await screen.findByText(/cannot support that answer/i),
    ).toBeInTheDocument();
  });

  it("presents a recoverable deterministic failure", async () => {
    const user = userEvent.setup();
    const corpus = corpora.at(0);
    if (!corpus) {
      throw new Error("Expected the Brazilian corpus fixture to be present.");
    }
    renderAtRoute(<ResearchChat corpus={corpus} onOpenCitation={vi.fn()} />);

    await user.type(
      screen.getByRole("textbox", { name: /question for this corpus/i }),
      "Simulate a failed response",
    );
    await user.click(screen.getByRole("button", { name: /send question/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      /could not be completed/i,
    );
  });
});
