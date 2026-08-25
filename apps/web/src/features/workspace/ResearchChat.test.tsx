import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { ChatProvider, RetrievalStrategy } from "../../api/chat";
import {
  expectNoAccessibilityViolations,
  renderAtRoute,
} from "../../test/render";
import { ResearchChat } from "./ResearchChat";

describe("research chat", () => {
  it("shows a localized thinking status before the first answer token", async () => {
    let completeRequest: (() => void) | undefined;
    const provider: ChatProvider = {
      streamQuestion: () =>
        new Promise<void>((resolve) => {
          completeRequest = resolve;
        }),
    };
    const result = renderAtRoute(
      <ResearchChat corpusId="corpus-1" provider={provider} />,
    );

    fireEvent.change(
      screen.getByRole("textbox", { name: "Research question" }),
      { target: { value: "What applies?" } },
    );
    fireEvent.submit(
      screen.getByRole("textbox", { name: "Research question" }),
    );

    expect(await screen.findByRole("status")).toHaveTextContent("Thinking");

    result.unmount();
    completeRequest?.();
  });

  it("submits with Enter and preserves a newline with Shift+Enter", async () => {
    const questions: string[] = [];
    const provider: ChatProvider = {
      streamQuestion: (
        _corpus,
        question,
        _language,
        _strategy,
        _signal,
        onEvent,
      ) => {
        questions.push(question);
        onEvent({
          type: "completed",
          requestId: "request-1",
          answer: "Completed.",
          references: [],
          telemetry: {
            outcome: "completed",
            evidenceCount: 0,
            durationMilliseconds: 1,
          },
        });
        return Promise.resolve();
      },
    };
    const user = userEvent.setup();
    renderAtRoute(<ResearchChat corpusId="corpus-1" provider={provider} />);

    const input = screen.getByRole("textbox", { name: "Research question" });
    await user.type(input, "First line{Shift>}{Enter}{/Shift}Second line");

    expect(input).toHaveValue("First line\nSecond line");

    await user.keyboard("{Enter}");

    expect(await screen.findByText("Completed.")).toBeVisible();
    expect(questions).toEqual(["First line\nSecond line"]);
  });

  it("uses the selected retrieval strategy for the next question", async () => {
    const strategies: RetrievalStrategy[] = [];
    const provider: ChatProvider = {
      streamQuestion: (
        _corpus,
        _question,
        _language,
        strategy,
        _signal,
        onEvent,
      ) => {
        strategies.push(strategy);
        onEvent({
          type: "completed",
          requestId: "request-strategy",
          answer: "Completed.",
          references: [],
          telemetry: {
            outcome: "completed",
            evidenceCount: 0,
            durationMilliseconds: 1,
          },
        });
        return Promise.resolve();
      },
    };
    const user = userEvent.setup();
    renderAtRoute(<ResearchChat corpusId="corpus-1" provider={provider} />);

    await user.click(screen.getByRole("radio", { name: "Hybrid" }));
    await user.type(
      screen.getByRole("textbox", { name: "Research question" }),
      "What applies?",
    );
    await user.keyboard("{Enter}");

    expect(strategies).toEqual(["hybrid"]);
  });

  it("submits a starter question through the normal chat runtime", async () => {
    const questions: string[] = [];
    const provider: ChatProvider = {
      streamQuestion: (
        _corpus,
        question,
        _language,
        _strategy,
        _signal,
        onEvent,
      ) => {
        questions.push(question);
        onEvent({
          type: "completed",
          requestId: "request-1",
          answer: "Completed.",
          references: [],
          telemetry: {
            outcome: "completed",
            evidenceCount: 0,
            durationMilliseconds: 1,
          },
        });
        return Promise.resolve();
      },
    };
    const user = userEvent.setup();
    renderAtRoute(<ResearchChat corpusId="corpus-1" provider={provider} />);

    await user.click(
      screen.getByRole("button", {
        name: "What is the purpose of this document?",
      }),
    );

    expect(await screen.findByText("Completed.")).toBeVisible();
    expect(questions).toEqual(["What is the purpose of this document?"]);
  });

  it("submits a corpus-scoped question and renders the grounded answer", async () => {
    const reference = {
      id: "reference-1",
      corpusId: "corpus-1",
      sourceId: "source-1",
      documentId: "document-1",
      documentVersionId: "document-1",
      sourceTitle: "Official GDPR text",
      unitLocator: "Article 1",
      startOffset: 0,
      endOffset: 10,
      excerpt: "Protected rights",
      rank: 1,
    };
    const provider: ChatProvider = {
      streamQuestion: (
        _corpus,
        _question,
        _language,
        _strategy,
        _signal,
        onEvent,
      ) => {
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
    expect(screen.getByRole("article", { name: "You" })).toBeVisible();
    expect(screen.getByRole("article", { name: "Norvii" })).toBeVisible();
    expect(screen.getByText("Protect rights").tagName).toBe("STRONG");
    expect(screen.getAllByRole("listitem")).toHaveLength(2);
    fireEvent.click(
      screen.getByRole("button", {
        name: "Open Article 1 in Official GDPR text",
      }),
    );
    expect(onReferenceSelect).toHaveBeenCalledWith(reference);
    await expectNoAccessibilityViolations(result.container);
  });

  it("groups repeated legal locations and reveals additional locations on demand", async () => {
    const references = [
      createReference(1, "Article 474"),
      createReference(2, "article-474"),
      createReference(3, "Article 3"),
      createReference(4, "Article 215"),
      createReference(5, "Article 416"),
      createReference(6, "Article 65"),
    ];
    const provider: ChatProvider = {
      streamQuestion: (
        _corpus,
        _question,
        _language,
        _strategy,
        _signal,
        onEvent,
      ) => {
        onEvent({
          type: "completed",
          requestId: "request-1",
          answer: "The rule applies [1] [2] [3] [4] [5] [6].",
          references,
          telemetry: {
            outcome: "completed",
            evidenceCount: references.length,
            durationMilliseconds: 1,
          },
        });
        return Promise.resolve();
      },
    };
    const onReferenceSelect = vi.fn();
    const user = userEvent.setup();
    renderAtRoute(
      <ResearchChat
        corpusId="corpus-1"
        provider={provider}
        onReferenceSelect={onReferenceSelect}
      />,
    );

    await user.type(
      screen.getByRole("textbox", { name: "Research question" }),
      "Which locations apply?",
    );
    await user.click(screen.getByRole("button", { name: "Send question" }));

    expect(
      await screen.findByRole("button", {
        name: "Open Article 474 in Official LGPD text",
      }),
    ).toHaveTextContent("2 supporting passages");
    expect(
      screen.getByRole("button", {
        name: "Open Article 215 in Official LGPD text",
      }),
    ).toBeVisible();
    expect(
      screen.queryByRole("button", {
        name: "Open Article 416 in Official LGPD text",
      }),
    ).not.toBeInTheDocument();

    await user.click(
      screen.getByRole("button", { name: "View 2 more locations" }),
    );

    await user.click(
      screen.getByRole("button", {
        name: "Open Article 416 in Official LGPD text",
      }),
    );
    expect(onReferenceSelect).toHaveBeenCalledWith(references[4]);
  });

  it("renders an abstention returned by grounded retrieval", async () => {
    const provider: ChatProvider = {
      streamQuestion: (
        _corpus,
        _question,
        _language,
        _strategy,
        _signal,
        onEvent,
      ) => {
        onEvent({
          type: "abstained",
          requestId: "request-1",
          reason: "insufficient evidence",
          telemetry: {
            outcome: "abstained",
            evidenceCount: 0,
            durationMilliseconds: 1,
          },
        });
        return Promise.resolve();
      },
    };
    renderAtRoute(<ResearchChat corpusId="corpus-1" provider={provider} />);

    fireEvent.change(
      screen.getByRole("textbox", { name: "Research question" }),
      {
        target: { value: "What is not covered?" },
      },
    );
    fireEvent.submit(
      screen.getByRole("textbox", { name: "Research question" }),
    );

    expect(
      await screen.findByText(
        "I could not find enough published evidence in this corpus to answer safely.",
      ),
    ).toBeVisible();
  });

  it("renders a compact research record without duplicating preserved evidence excerpts", async () => {
    const reference = {
      id: "reference-1",
      corpusId: "corpus-1",
      sourceId: "source-1",
      documentId: "document-1",
      documentVersionId: "document-1",
      sourceRevisionId: "revision-1",
      pipelineVersion: "corpus-ingestion-v1",
      sourceTitle: "Official GDPR text",
      cosineDistance: null,
      unitLocator: "Article 1",
      startOffset: 0,
      endOffset: 10,
      excerpt: "Protected rights",
      rank: 1,
    };
    const provider: ChatProvider = {
      streamQuestion: (
        _corpus,
        _question,
        _language,
        _strategy,
        _signal,
        onEvent,
      ) => {
        onEvent({
          type: "completed",
          requestId: "request-1",
          answer: "The rule applies [1].",
          references: [reference],
          telemetry: {
            outcome: "completed",
            evidenceCount: 1,
            durationMilliseconds: 12,
          },
          inspection: {
            outcome: "completed",
            retrieval: {
              strategy: "vector",
              topK: 8,
              returnedCount: 1,
              embeddingModel: null,
            },
            measurements: {
              retrievalMilliseconds: 2,
              generationMilliseconds: 9,
              totalMilliseconds: 12,
              inputTokens: null,
              outputTokens: null,
            },
            evidence: [reference],
          },
        });
        return Promise.resolve();
      },
    };
    const onReferenceSelect = vi.fn();
    const user = userEvent.setup();
    renderAtRoute(
      <ResearchChat
        corpusId="corpus-1"
        provider={provider}
        onReferenceSelect={onReferenceSelect}
      />,
    );

    await user.type(
      screen.getByRole("textbox", { name: "Research question" }),
      "What applies?",
    );
    await user.click(screen.getByRole("button", { name: "Send question" }));
    expect(await screen.findByText("Research record")).toBeVisible();
    expect(screen.queryByText("Inspect answer")).not.toBeInTheDocument();
    expect(screen.getAllByText("Unavailable")[0]).not.toBeVisible();
    await user.click(screen.getByText("Research record"));

    expect(screen.getAllByText("Unavailable")).toHaveLength(2);
    expect(screen.queryByText("Protected rights")).not.toBeInTheDocument();
    await user.click(
      within(
        screen.getByRole("list", { name: "Supporting passages" }),
      ).getByRole("button", { name: /Official GDPR text\s*Article 1/ }),
    );
    expect(onReferenceSelect).toHaveBeenCalledWith(reference);
  });

  it("shows stream errors and provider failures to the researcher", async () => {
    const streamErrorProvider: ChatProvider = {
      streamQuestion: (
        _corpus,
        _question,
        _language,
        _strategy,
        _signal,
        onEvent,
      ) => {
        onEvent({
          type: "error",
          requestId: "request-1",
          code: "provider_error",
          message: "The upstream provider failed.",
          telemetry: {
            outcome: "error",
            evidenceCount: 0,
            durationMilliseconds: 1,
          },
        });
        return Promise.resolve();
      },
    };
    const streamErrorResult = renderAtRoute(
      <ResearchChat corpusId="corpus-1" provider={streamErrorProvider} />,
    );
    fireEvent.change(
      screen.getByRole("textbox", { name: "Research question" }),
      { target: { value: "Will this fail?" } },
    );
    fireEvent.submit(
      screen.getByRole("textbox", { name: "Research question" }),
    );
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "The upstream provider failed.",
    );
    streamErrorResult.unmount();

    const rejectedProvider: ChatProvider = {
      streamQuestion: () => Promise.reject(new Error("Network unavailable")),
    };
    renderAtRoute(
      <ResearchChat corpusId="corpus-1" provider={rejectedProvider} />,
    );
    fireEvent.change(
      screen.getByRole("textbox", { name: "Research question" }),
      { target: { value: "Will the network fail?" } },
    );
    fireEvent.submit(
      screen.getByRole("textbox", { name: "Research question" }),
    );
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Network unavailable",
    );
  });

  it("explains when the selected snapshot has no graph release", async () => {
    const provider: ChatProvider = {
      streamQuestion: (
        _corpus,
        _question,
        _language,
        _strategy,
        _signal,
        onEvent,
      ) => {
        onEvent({
          type: "error",
          requestId: "request-graph",
          code: "graph_unavailable",
          message: "The grounded chat request could not be completed.",
          telemetry: {
            outcome: "failed",
            evidenceCount: 0,
            durationMilliseconds: 1,
          },
        });
        return Promise.resolve();
      },
    };
    renderAtRoute(<ResearchChat corpusId="corpus-1" provider={provider} />);

    fireEvent.change(
      screen.getByRole("textbox", { name: "Research question" }),
      { target: { value: "What applies?" } },
    );
    fireEvent.submit(
      screen.getByRole("textbox", { name: "Research question" }),
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Graph and hybrid retrieval are unavailable for this snapshot",
    );
  });

  it("compares independent retrieval strategies without masking an unavailable graph", async () => {
    const strategies: string[] = [];
    const provider: ChatProvider = {
      streamQuestion: (
        _corpus,
        _question,
        _language,
        strategy,
        _signal,
        onEvent,
      ) => {
        strategies.push(strategy);
        if (strategy === "graph") {
          onEvent({
            type: "error",
            requestId: `request-${strategy}`,
            code: "graph_unavailable",
            message: "The graph release is unavailable.",
            telemetry: {
              outcome: "failed",
              evidenceCount: 0,
              durationMilliseconds: 1,
            },
          });
          return Promise.resolve();
        }
        onEvent({
          type: "completed",
          requestId: `request-${strategy}`,
          answer: `${strategy} answer`,
          references: [],
          telemetry: {
            outcome: "completed",
            evidenceCount: 0,
            durationMilliseconds: 1,
          },
          inspection: {
            outcome: "completed",
            retrieval: {
              strategy,
              topK: 8,
              returnedCount: 0,
              embeddingModel: null,
            },
            measurements: {
              retrievalMilliseconds: 1,
              generationMilliseconds: 1,
              totalMilliseconds: 2,
              inputTokens: 1,
              outputTokens: 1,
            },
          },
        });
        return Promise.resolve();
      },
    };
    const user = userEvent.setup();
    renderAtRoute(<ResearchChat corpusId="corpus-1" provider={provider} />);

    await user.type(
      screen.getByRole("textbox", { name: "Research question" }),
      "What applies?",
    );
    await user.click(screen.getByRole("button", { name: "Send question" }));
    await user.click(
      await screen.findByRole("button", { name: "Compare strategies" }),
    );

    expect(await screen.findAllByText("vector answer")).toHaveLength(2);
    expect(screen.getByText("hybrid answer")).toBeVisible();
    expect(screen.getByText("Unavailable")).toBeVisible();
    expect(strategies).toEqual(["vector", "vector", "graph", "hybrid"]);
  });

  it("cancels all comparison requests when its chat view unmounts", async () => {
    const comparisonSignals: AbortSignal[] = [];
    let requestCount = 0;
    const provider: ChatProvider = {
      streamQuestion: (
        _corpus,
        _question,
        _language,
        _strategy,
        signal,
        onEvent,
      ) => {
        requestCount += 1;
        if (requestCount === 1) {
          onEvent({
            type: "completed",
            requestId: "normal-request",
            answer: "Normal answer.",
            references: [],
            telemetry: {
              outcome: "completed",
              evidenceCount: 0,
              durationMilliseconds: 1,
            },
          });
          return Promise.resolve();
        }
        comparisonSignals.push(signal);
        return new Promise<void>(() => undefined);
      },
    };
    const user = userEvent.setup();
    const result = renderAtRoute(
      <ResearchChat corpusId="corpus-1" provider={provider} />,
    );

    await user.type(
      screen.getByRole("textbox", { name: "Research question" }),
      "What applies?",
    );
    await user.click(screen.getByRole("button", { name: "Send question" }));
    await user.click(
      await screen.findByRole("button", { name: "Compare strategies" }),
    );
    await waitFor(() => expect(comparisonSignals).toHaveLength(3));

    result.unmount();

    expect(comparisonSignals.every((signal) => signal.aborted)).toBe(true);
  });
});

function createReference(rank: number, unitLocator: string) {
  return {
    id: `reference-${String(rank)}`,
    corpusId: "corpus-1",
    sourceId: "source-1",
    documentId: "document-1",
    documentVersionId: "document-1",
    sourceTitle: "Official LGPD text",
    unitLocator,
    startOffset: rank * 10,
    endOffset: rank * 10 + 5,
    excerpt: "Protected legal text.",
    rank,
  };
}
