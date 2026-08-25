import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type {
  ChatProvider,
  ChatReference,
  RetrievalStrategy,
} from "../../api/chat";
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

  it("stops the active request and records the cancellation in the conversation", async () => {
    let requestWasAborted = false;
    const provider: ChatProvider = {
      streamQuestion: (_corpus, _question, _language, _strategy, signal) =>
        new Promise<void>((resolve) => {
          signal.addEventListener(
            "abort",
            () => {
              requestWasAborted = true;
              resolve();
            },
            { once: true },
          );
        }),
    };
    const user = userEvent.setup();
    renderAtRoute(<ResearchChat corpusId="corpus-1" provider={provider} />);

    await user.type(
      screen.getByRole("textbox", { name: "Research question" }),
      "What applies?",
    );
    await user.keyboard("{Enter}");

    await user.click(
      await screen.findByRole("button", { name: "Stop response" }),
    );

    expect(await screen.findByText("Response cancelled.")).toBeVisible();
    expect(requestWasAborted).toBe(true);
    expect(screen.queryByText("Thinking")).not.toBeInTheDocument();
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

    await user.click(screen.getByRole("radio", { name: "Vector" }));
    await user.type(
      screen.getByRole("textbox", { name: "Research question" }),
      "What applies?",
    );
    await user.keyboard("{Enter}");

    expect(strategies).toEqual(["vector"]);
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

  it("offers graph-oriented starter questions for the demonstration", async () => {
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
          requestId: "graph-starter-question",
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

    const authorityReportsQuestion = await screen.findByRole("button", {
      name: "Which reports does the national data protection authority require?",
    });
    expect(authorityReportsQuestion).toBeVisible();
    expect(
      screen.getByRole("button", {
        name: "What does this document require the data protection authority to do?",
      }),
    ).toBeVisible();

    expect(
      screen.getByRole("button", {
        name: "Which rights does this document grant to data subjects?",
      }),
    ).toBeVisible();

    await user.click(authorityReportsQuestion);

    expect(strategies).toEqual(["hybrid"]);
  });

  it("submits a corpus-scoped question and renders the grounded answer", async () => {
    const reference: ChatReference = {
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

  it("shows every unique legal location with its retrieval contribution", async () => {
    const references: readonly ChatReference[] = [
      { ...createReference(1, "Article 474"), contribution: "vector" },
      { ...createReference(2, "article-474"), contribution: "graph" },
      { ...createReference(3, "Article 3"), contribution: "vector" },
      { ...createReference(4, "Article 215"), contribution: "graph" },
      { ...createReference(5, "Article 416"), contribution: "graph" },
      { ...createReference(6, "Article 65"), contribution: "vector" },
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
    ).toHaveTextContent("Vector and graph evidence");
    expect(
      screen.getByRole("button", {
        name: "Open Article 416 in Official LGPD text",
      }),
    ).toHaveTextContent("Graph evidence");
    expect(
      screen.queryByRole("button", { name: /View .*more locations/ }),
    ).not.toBeInTheDocument();

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
    const reference: ChatReference = {
      id: "reference-1",
      corpusId: "corpus-1",
      sourceId: "source-1",
      documentId: "document-1",
      documentVersionId: "document-1",
      sourceRevisionId: "revision-1",
      pipelineVersion: "corpus-ingestion-v1",
      sourceTitle: "Official GDPR text",
      contribution: "vector",
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
            stages: [
              {
                name: "vector",
                state: "completed",
                evidenceCount: 1,
                durationMilliseconds: 2,
                reasonCode: null,
                inputTokens: null,
                outputTokens: null,
              },
              {
                name: "planning",
                state: "skipped",
                evidenceCount: 0,
                durationMilliseconds: 1,
                reasonCode: "not_relevant",
                inputTokens: 8,
                outputTokens: 1,
              },
              {
                name: "graph",
                state: "skipped",
                evidenceCount: 0,
                durationMilliseconds: null,
                reasonCode: "not_relevant",
                inputTokens: null,
                outputTokens: null,
              },
            ],
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

    expect(screen.getByText("Vector retrieval")).toBeVisible();
    expect(screen.getByText("Graph planning")).toBeVisible();
    expect(screen.getAllByText(/Not relevant to this question/)).toHaveLength(
      2,
    );
    expect(screen.getAllByText(/Vector evidence/)).toHaveLength(2);
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
    expect(screen.getByRole("article", { name: "Norvii" })).toContainElement(
      screen.getByRole("alert"),
    );
    expect(screen.queryByText("Thinking")).not.toBeInTheDocument();
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

  it("compares Vector and Hybrid retrieval strategies", async () => {
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
    const compareButton = await screen.findByRole("button", {
      name: "Compare strategies",
    });

    expect(compareButton.closest(".chat-viewport__footer")).toBeNull();
    expect(compareButton.closest(".chat-viewport")).not.toBeNull();

    await user.click(compareButton);

    expect(await screen.findByText("vector answer")).toBeVisible();
    expect(screen.getAllByText("hybrid answer")).toHaveLength(2);
    expect(strategies).toEqual(["hybrid", "vector", "hybrid"]);

    await user.type(
      screen.getByRole("textbox", { name: "Research question" }),
      "Which authority is responsible?",
    );
    await user.click(screen.getByRole("button", { name: "Send question" }));

    expect(screen.queryByText("vector answer")).not.toBeInTheDocument();
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
    await waitFor(() => expect(comparisonSignals).toHaveLength(2));

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
