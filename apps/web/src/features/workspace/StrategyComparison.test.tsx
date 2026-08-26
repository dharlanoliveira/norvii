import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type {
  ChatProvider,
  ChatReference,
  ChatStreamEvent,
  RetrievalStrategy,
} from "../../api/chat";
import { renderAtRoute } from "../../test/render";
import { StrategyComparison } from "./StrategyComparison";

describe("strategy comparison", () => {
  it("compares Vector and Hybrid and opens cited evidence and graph paths", async () => {
    const user = userEvent.setup();
    const onReferenceSelect = vi.fn();
    renderAtRoute(
      <StrategyComparison
        corpusId="corpus-1"
        interfaceLanguage="en"
        onReferenceSelect={onReferenceSelect}
        provider={completingProvider()}
        question="Which provision applies?"
      />,
    );

    await user.click(
      screen.getByRole("button", { name: "Compare strategies" }),
    );

    expect(await screen.findByText("vector answer")).toBeVisible();
    expect(screen.getByText("hybrid answer")).toBeVisible();

    await user.click(screen.getByRole("button", { name: "vector Article 1" }));
    expect(onReferenceSelect).toHaveBeenCalledWith(reference("vector"));

    await user.click(
      screen.getByRole("button", {
        name: "vector scope / applies to / Authority",
      }),
    );
    expect(onReferenceSelect).toHaveBeenCalledTimes(2);
  });

  it("reports failed strategies when their requests cannot start", async () => {
    const user = userEvent.setup();
    const provider: ChatProvider = {
      streamQuestion: () => Promise.reject(new Error("unavailable")),
    };
    renderAtRoute(
      <StrategyComparison
        corpusId="corpus-1"
        interfaceLanguage="en"
        provider={provider}
        question="Which provision applies?"
      />,
    );

    await user.click(
      screen.getByRole("button", { name: "Compare strategies" }),
    );

    expect(await screen.findAllByText("Failed")).toHaveLength(2);
  });
});

function completingProvider(): ChatProvider {
  return {
    streamQuestion: (
      _corpusId,
      _question,
      _language,
      strategy,
      _signal,
      onEvent,
    ) => {
      onEvent(completedEvent(strategy));
      return Promise.resolve();
    },
  };
}

function completedEvent(strategy: RetrievalStrategy): ChatStreamEvent {
  const citedReference = reference(strategy);
  return {
    type: "completed" as const,
    requestId: `request-${strategy}`,
    answer: `${strategy} answer`,
    references: [citedReference],
    telemetry: {
      outcome: "completed",
      evidenceCount: 1,
      durationMilliseconds: 1,
    },
    inspection: {
      outcome: "completed" as const,
      measurements: {
        retrievalMilliseconds: 1,
        generationMilliseconds: 2,
        totalMilliseconds: 3,
        inputTokens: 4,
        outputTokens: 5,
      },
      assertionPath: [
        {
          assertionId: `assertion-${strategy}`,
          predicate: "applies_to",
          evidenceLocator: citedReference.unitLocator,
          establishingLocator: citedReference.unitLocator,
          subjectLabel: `${strategy} scope`,
          objectLabel: "Authority",
          hierarchyContext: [citedReference.unitLocator],
          qualifier: null,
        },
      ],
    },
  };
}

function reference(strategy: RetrievalStrategy): ChatReference {
  return {
    id: "reference-1",
    corpusId: "corpus-1",
    sourceId: "source-1",
    documentId: "document-1",
    documentVersionId: "document-version-1",
    unitLocator: `${strategy} Article 1`,
    startOffset: 0,
    endOffset: 10,
    excerpt: "Scope provision",
    rank: 1,
  };
}
