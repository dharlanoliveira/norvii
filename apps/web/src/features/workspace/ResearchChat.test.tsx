import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { ChatProvider } from "../../api/chat";
import {
  expectNoAccessibilityViolations,
  renderAtRoute,
} from "../../test/render";
import { ResearchChat } from "./ResearchChat";

describe("research chat", () => {
  it("submits a corpus-scoped question and renders the grounded answer", async () => {
    const reference = {
      id: "reference-1",
      corpusId: "corpus-1",
      sourceId: "source-1",
      documentId: "document-1",
      unitLocator: "Article 1",
      startOffset: 0,
      endOffset: 10,
      excerpt: "Protected rights",
      rank: 1,
    };
    const provider: ChatProvider = {
      streamQuestion: (_corpus, _question, _language, _signal, onEvent) => {
        onEvent({
          type: "started",
          requestId: "request-1",
          corpusId: "corpus-1",
        });
        onEvent({
          type: "evidence",
          requestId: "request-1",
          references: [reference],
        });
        onEvent({
          type: "delta",
          requestId: "request-1",
          text: "The rule applies ",
        });
        onEvent({
          type: "completed",
          requestId: "request-1",
          answer:
            "## Purpose\n\n- **Protect rights** for data subjects [1].\n- Enable lawful data movement [2].",
          references: [reference],
          telemetry: {
            outcome: "completed",
            evidenceCount: 0,
            durationMilliseconds: 1,
          },
        });
        return Promise.resolve();
      },
    };
    const onReferenceSelect = vi.fn();
    const result = renderAtRoute(
      <ResearchChat
        corpusId="corpus-1"
        onReferenceSelect={onReferenceSelect}
        provider={provider}
      />,
    );

    fireEvent.change(
      screen.getByRole("textbox", { name: "Research question" }),
      {
        target: { value: "What applies?" },
      },
    );
    fireEvent.submit(
      screen.getByRole("textbox", { name: "Research question" }),
    );

    expect(
      await screen.findByRole("heading", { name: "Purpose" }),
    ).toBeVisible();
    expect(screen.getByText("Protect rights").tagName).toBe("STRONG");
    expect(screen.getAllByRole("listitem")).toHaveLength(2);
    fireEvent.click(screen.getByRole("button", { name: "[1] Article 1" }));
    expect(onReferenceSelect).toHaveBeenCalledWith(reference);
    await expectNoAccessibilityViolations(result.container);
  });
});
