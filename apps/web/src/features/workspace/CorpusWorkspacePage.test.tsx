import { act, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { Route, Routes, useNavigate } from "react-router-dom";

import { createHttpResearchProvider } from "../../api/researchProvider";
import type { ChatProvider } from "../../api/chat";
import type {
  ActiveSnapshotResponse,
  CorpusResponse,
} from "../../api/contract";
import { i18n } from "../../i18n/config";
import type { ResearchProvider } from "../../research/domain/authoritative";
import { renderAtRoute } from "../../test/render";
import { CorpusWorkspacePage } from "./CorpusWorkspacePage";

describe("authoritative corpus workspace", () => {
  it("loads corpus-owned sources and opens a persisted document", async () => {
    const fetchResponse = vi.fn<typeof fetch>().mockImplementation((input) => {
      const url =
        typeof input === "string"
          ? input
          : input instanceof URL
            ? input.href
            : input.url;
      if (url.endsWith("/sources"))
        return Promise.resolve(jsonResponse([source()]));
      if (url.endsWith("/document") || url.includes("/documents/"))
        return Promise.resolve(jsonResponse(document()));
      return Promise.resolve(jsonResponse(corpus()));
    });
    const provider = createHttpResearchProvider({ fetch: fetchResponse });
    const chatProvider: ChatProvider = {
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
          answer: "The answer is grounded. [1]",
          references: [
            {
              id: "reference-1",
              corpusId: "10000000-0000-4000-8000-000000000002",
              sourceId: "20000000-0000-4000-8000-000000000002",
              documentId: "50000000-0000-4000-8000-000000000001",
              documentVersionId: "50000000-0000-4000-8000-000000000001",
              sourceTitle: "Official English GDPR text",
              unitLocator: "article-1",
              startOffset: 0,
              endOffset: 21,
              excerpt: "Persisted legal text.",
              rank: 1,
            },
          ],
          telemetry: {
            outcome: "completed",
            evidenceCount: 1,
            durationMilliseconds: 1,
          },
        });
        return Promise.resolve();
      },
    };
    const user = userEvent.setup();

    renderAtRoute(
      <Routes>
        <Route
          path="corpora/:corpusId"
          element={
            <CorpusWorkspacePage
              provider={provider}
              chatProvider={chatProvider}
            />
          }
        />
      </Routes>,
      "/corpora/10000000-0000-4000-8000-000000000002",
    );

    await screen.findByRole("treeitem", {
      name: /Official English GDPR text/,
    });
    await user.click(screen.getByRole("tab", { name: "Source" }));
    expect(
      screen.getByRole("heading", {
        name: "Open a source to begin reviewing.",
      }),
    ).toBeVisible();
    expect(
      screen.getByText(
        "Inspect preserved legal text and open cited provisions while keeping the conversation available.",
      ),
    ).toBeVisible();
    await user.click(
      screen.getByRole("button", {
        name: "Open Official English GDPR text",
      }),
    );
    expect(await screen.findByText("Persisted legal text.")).toBeVisible();
    await user.click(screen.getByRole("tab", { name: "Chat" }));
    await user.type(
      screen.getByRole("textbox", { name: "Research question" }),
      "Which article applies?",
    );
    await user.click(screen.getByRole("button", { name: "Send question" }));
    await user.click(
      await screen.findByRole("button", {
        name: "Open Article 1 in Official English GDPR text",
      }),
    );

    expect(await screen.findByText("Persisted legal text.")).toBeVisible();
    const immutableDocumentCall = fetchResponse.mock.calls.find(([input]) => {
      const url =
        typeof input === "string"
          ? input
          : input instanceof URL
            ? input.href
            : input.url;
      return url.includes("/documents/50000000-0000-4000-8000-000000000001");
    });
    expect(immutableDocumentCall).toBeDefined();
    expect(immutableDocumentCall?.[1]?.signal).toBeInstanceOf(AbortSignal);
    expect(
      fetchResponse.mock.calls.some(([input]) => {
        const url =
          typeof input === "string"
            ? input
            : input instanceof URL
              ? input.href
              : input.url;
        return url.endsWith("/document");
      }),
    ).toBe(true);
    expect(screen.getByRole("article", { name: "Article 1" })).toHaveAttribute(
      "data-selected",
      "true",
    );
  });

  it("does not cross-load another corpus when the workspace request fails", async () => {
    const fetchResponse = vi.fn<typeof fetch>().mockResolvedValue(
      new Response(JSON.stringify({ error: { code: "not_found" } }), {
        status: 404,
      }),
    );

    renderAtRoute(
      <Routes>
        <Route
          path="corpora/:corpusId"
          element={
            <CorpusWorkspacePage
              provider={createHttpResearchProvider({ fetch: fetchResponse })}
            />
          }
        />
      </Routes>,
      "/corpora/10000000-0000-4000-8000-000000000099",
    );

    expect(await screen.findByRole("alert")).toBeVisible();
    expect(screen.queryByRole("tree")).not.toBeInTheDocument();
  });

  it("shows an explicit source-empty state with both add actions", async () => {
    const user = userEvent.setup();
    const fetchResponse = vi.fn<typeof fetch>().mockImplementation((input) => {
      const url =
        typeof input === "string"
          ? input
          : input instanceof URL
            ? input.href
            : input.url;
      return Promise.resolve(
        jsonResponse(url.endsWith("/sources") ? [] : corpus()),
      );
    });
    renderAtRoute(
      <Routes>
        <Route
          path="corpora/:corpusId"
          element={
            <CorpusWorkspacePage
              provider={createHttpResearchProvider({ fetch: fetchResponse })}
            />
          }
        />
      </Routes>,
      "/corpora/10000000-0000-4000-8000-000000000002",
    );

    expect(
      await screen.findByText("No sources are registered in this corpus yet."),
    ).toBeVisible();
    const addUrl = screen.getByRole("button", { name: "Add official URL" });
    const addPdf = screen.getByRole("button", { name: "Upload PDF" });
    expect(addUrl).toHaveAttribute("aria-expanded", "false");
    expect(addPdf).toHaveAttribute("aria-expanded", "false");

    await user.click(screen.getByRole("tab", { name: "Source" }));
    await user.click(
      screen.getByRole("button", { name: "Add an official source" }),
    );
    expect(addUrl).toHaveAttribute("aria-expanded", "true");
    expect(
      screen.getByRole("form", { name: "Add an official web source" }),
    ).toBeVisible();

    await user.click(addUrl);
    expect(addUrl).toHaveAttribute("aria-expanded", "false");

    await user.click(addPdf);
    expect(addUrl).toHaveAttribute("aria-expanded", "false");
    expect(addPdf).toHaveAttribute("aria-expanded", "true");
    expect(
      screen.getByRole("form", { name: "Upload an official PDF" }),
    ).toBeVisible();
  });

  it("registers an official URL source in the active corpus", async () => {
    const user = userEvent.setup();
    const fetchResponse = vi.fn<typeof fetch>().mockImplementation((input) => {
      const url = requestUrl(input);
      if (url.endsWith("/sources/url"))
        return Promise.resolve(jsonResponse(source()));
      if (url.endsWith("/sources")) return Promise.resolve(jsonResponse([]));
      return Promise.resolve(jsonResponse(corpus()));
    });

    renderAtRoute(
      <Routes>
        <Route
          path="corpora/:corpusId"
          element={
            <CorpusWorkspacePage
              provider={createHttpResearchProvider({ fetch: fetchResponse })}
            />
          }
        />
      </Routes>,
      "/corpora/10000000-0000-4000-8000-000000000002",
    );

    await screen.findByText("No sources are registered in this corpus yet.");
    await user.click(screen.getByRole("button", { name: "Add official URL" }));
    await user.type(
      screen.getByRole("textbox", { name: "Source title" }),
      "Official English GDPR text",
    );
    await user.type(
      screen.getByRole("textbox", { name: "Official HTTPS URL" }),
      "https://example.org/gdpr",
    );
    await user.click(screen.getByRole("button", { name: "Add URL source" }));

    expect(
      await screen.findByRole("treeitem", {
        name: /Official English GDPR text/,
      }),
    ).toBeVisible();
    const createCall = fetchResponse.mock.calls.find(([input]) =>
      requestUrl(input).endsWith("/sources/url"),
    );
    expect(createCall?.[1]).toMatchObject({ method: "POST" });
    const requestBody = createCall?.[1]?.body;
    expect(typeof requestBody).toBe("string");
    if (typeof requestBody !== "string")
      throw new Error("Expected URL source request body to be JSON.");
    expect(JSON.parse(requestBody)).toEqual({
      title: "Official English GDPR text",
      url: "https://example.org/gdpr",
    });
  });

  it("keeps a failed source selectable and retries its ingestion", async () => {
    const failedSource = {
      ...source(),
      processingStatus: "failed",
      failureCategory: "acquisition_failed",
      latestReadyDocumentId: null,
    };
    const retriedSource = {
      ...failedSource,
      processingStatus: "processing",
      failureCategory: null,
      version: 3,
    };
    const fetchResponse = vi.fn<typeof fetch>().mockImplementation((input) => {
      const url = requestUrl(input);
      if (url.endsWith("/retry"))
        return Promise.resolve(jsonResponse(retriedSource));
      if (url.endsWith("/sources"))
        return Promise.resolve(jsonResponse([failedSource]));
      return Promise.resolve(jsonResponse(corpus()));
    });
    const user = userEvent.setup();

    renderAtRoute(
      <Routes>
        <Route
          path="corpora/:corpusId"
          element={
            <CorpusWorkspacePage
              provider={createHttpResearchProvider({ fetch: fetchResponse })}
            />
          }
        />
      </Routes>,
      "/corpora/10000000-0000-4000-8000-000000000002",
    );

    await user.click(
      await screen.findByRole("treeitem", {
        name: "Official English GDPR text (Failed)",
      }),
    );
    expect(
      await screen.findByText(
        "The latest attempt ended with a safe failure category.",
      ),
    ).toBeVisible();
    await user.click(screen.getByRole("button", { name: "Retry ingestion" }));

    expect(
      await screen.findByText("Processing", { selector: "output" }),
    ).toBeVisible();
    expect(
      fetchResponse.mock.calls.some(([input]) =>
        requestUrl(input).endsWith("/retry"),
      ),
    ).toBe(true);
  });

  it("keeps the source visible when its current document cannot be loaded", async () => {
    const fetchResponse = vi.fn<typeof fetch>().mockImplementation((input) => {
      const url = requestUrl(input);
      if (url.endsWith("/sources"))
        return Promise.resolve(jsonResponse([source()]));
      if (url.endsWith("/document")) {
        return Promise.resolve(new Response(null, { status: 503 }));
      }
      return Promise.resolve(jsonResponse(corpus()));
    });
    const user = userEvent.setup();

    renderWorkspace(fetchResponse);

    await user.click(
      await screen.findByRole("treeitem", {
        name: "Official English GDPR text (Ready)",
      }),
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "The persisted document could not be loaded.",
    );
    expect(
      screen.getByRole("heading", { name: "Official English GDPR text" }),
    ).toBeVisible();
  });

  it("explains when a chat citation cannot resolve to a corpus source", async () => {
    const user = userEvent.setup();
    const chatProvider: ChatProvider = {
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
          answer: "A cited answer.",
          references: [
            {
              id: "missing-reference",
              corpusId: "10000000-0000-4000-8000-000000000002",
              sourceId: "missing-source",
              documentId: "missing-document",
              documentVersionId: "missing-version",
              sourceTitle: "Unavailable source",
              unitLocator: "Article 1",
              startOffset: 0,
              endOffset: 1,
              excerpt: "Unavailable.",
              rank: 1,
            },
          ],
          telemetry: {
            outcome: "completed",
            evidenceCount: 1,
            durationMilliseconds: 1,
          },
        });
        return Promise.resolve();
      },
    };
    const fetchResponse = vi.fn<typeof fetch>().mockImplementation((input) => {
      const url = requestUrl(input);
      return Promise.resolve(
        jsonResponse(url.endsWith("/sources") ? [source()] : corpus()),
      );
    });

    renderWorkspace(fetchResponse, chatProvider);

    await user.type(
      await screen.findByRole("textbox", { name: "Research question" }),
      "Which article applies?",
    );
    await user.click(screen.getByRole("button", { name: "Send question" }));
    await user.click(
      await screen.findByRole("button", {
        name: "Open Article 1 in Unavailable source",
      }),
    );
    await user.click(screen.getByRole("tab", { name: "Source" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "The cited location is unavailable.",
    );
  });

  it("rejects an immutable document that does not contain the cited location", async () => {
    const user = userEvent.setup();
    const chatProvider: ChatProvider = {
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
          answer: "A cited answer.",
          references: [
            {
              id: "reference-1",
              corpusId: "10000000-0000-4000-8000-000000000002",
              sourceId: "20000000-0000-4000-8000-000000000002",
              documentId: "50000000-0000-4000-8000-000000000001",
              documentVersionId: "50000000-0000-4000-8000-000000000001",
              sourceTitle: "Official English GDPR text",
              unitLocator: "Article 1",
              startOffset: 0,
              endOffset: 1,
              excerpt: "Scope.",
              rank: 1,
            },
          ],
          telemetry: {
            outcome: "completed",
            evidenceCount: 1,
            durationMilliseconds: 1,
          },
        });
        return Promise.resolve();
      },
    };
    const fetchResponse = vi.fn<typeof fetch>().mockImplementation((input) => {
      const url = requestUrl(input);
      if (url.endsWith("/sources"))
        return Promise.resolve(jsonResponse([source()]));
      if (url.includes("/documents/")) {
        return Promise.resolve(
          jsonResponse({
            ...document(),
            id: "50000000-0000-4000-8000-000000000099",
          }),
        );
      }
      return Promise.resolve(jsonResponse(corpus()));
    });

    renderWorkspace(fetchResponse, chatProvider);

    await user.type(
      await screen.findByRole("textbox", { name: "Research question" }),
      "Which article applies?",
    );
    await user.click(screen.getByRole("button", { name: "Send question" }));
    await user.click(
      await screen.findByRole("button", {
        name: "Open Article 1 in Official English GDPR text",
      }),
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "The cited location is unavailable.",
    );
  });

  it("renders only exact rank-ordered questions in the active interface language", async () => {
    const snapshot = activeSnapshot();
    const englishResponse = deferred<Response>();
    let englishSignal: AbortSignal | undefined;
    const fetchResponse = vi
      .fn<typeof fetch>()
      .mockImplementation((input, init) => {
        const url = requestUrl(input);
        if (url.includes("/opening-suggestions?")) {
          if (url.endsWith("interfaceLanguage=en")) {
            englishSignal = init?.signal ?? undefined;
            return englishResponse.promise;
          }
          return Promise.resolve(
            jsonResponse(
              openingSuggestionsResponse(corpus().id, "pt", snapshot, [
                { caseId: "question-2", rank: 2, question: "Localized two" },
                { caseId: "question-5", rank: 5, question: "Localized five" },
              ]),
            ),
          );
        }
        if (url.endsWith("/sources")) return Promise.resolve(jsonResponse([]));
        return Promise.resolve(jsonResponse(corpusWithSnapshot(snapshot)));
      });

    renderWorkspace(fetchResponse);

    await waitFor(() => expect(englishSignal).toBeInstanceOf(AbortSignal));
    await act(async () => {
      await i18n.changeLanguage("pt");
    });

    expect(englishSignal?.aborted).toBe(true);
    expect(
      fetchResponse.mock.calls.some(([input]) =>
        requestUrl(input).endsWith("interfaceLanguage=pt"),
      ),
    ).toBe(true);
    englishResponse.resolve(
      jsonResponse(
        openingSuggestionsResponse(corpus().id, "en", snapshot, [
          { caseId: "question-1", rank: 1, question: "Stale English question" },
        ]),
      ),
    );

    const suggestions = await screen.findByLabelText(
      i18n.t("chat.starterQuestionsLabel"),
    );
    expect(
      within(suggestions)
        .getAllByRole("button")
        .map((button) => button.textContent),
    ).toEqual(["Localized two", "Localized five"]);
    expect(
      screen.queryByText("Stale English question"),
    ).not.toBeInTheDocument();
  });

  it("does not substitute a localized question when the suggestion projection fails", async () => {
    const snapshot = activeSnapshot();
    const fetchResponse = vi.fn<typeof fetch>().mockImplementation((input) => {
      const url = requestUrl(input);
      if (url.includes("/opening-suggestions?")) {
        return Promise.resolve(new Response(null, { status: 503 }));
      }
      if (url.endsWith("/sources")) return Promise.resolve(jsonResponse([]));
      return Promise.resolve(jsonResponse(corpusWithSnapshot(snapshot)));
    });

    renderWorkspace(fetchResponse);

    await waitFor(() =>
      expect(
        fetchResponse.mock.calls.some(([input]) =>
          requestUrl(input).includes("/opening-suggestions?"),
        ),
      ).toBe(true),
    );
    expect(
      screen.queryByLabelText("Suggested research questions"),
    ).not.toBeInTheDocument();
  });

  it("aborts and discards a pending projection when the corpus route changes", async () => {
    const firstCorpusId = corpus().id;
    const secondCorpusId = "10000000-0000-4000-8000-000000000003";
    const firstSnapshot = activeSnapshot();
    const secondSnapshot = activeSnapshot(
      "70000000-0000-4000-8000-000000000002",
    );
    const firstResponse = deferred<Response>();
    let firstSignal: AbortSignal | undefined;
    const fetchResponse = vi
      .fn<typeof fetch>()
      .mockImplementation((input, init) => {
        const url = requestUrl(input);
        if (url.includes("/opening-suggestions?")) {
          if (url.includes(firstCorpusId)) {
            firstSignal = init?.signal ?? undefined;
            return firstResponse.promise;
          }
          return Promise.resolve(
            jsonResponse(
              openingSuggestionsResponse(secondCorpusId, "en", secondSnapshot, [
                {
                  caseId: "second-corpus-question",
                  rank: 1,
                  question: "Second corpus question",
                },
              ]),
            ),
          );
        }
        if (url.endsWith("/sources")) return Promise.resolve(jsonResponse([]));
        return Promise.resolve(
          jsonResponse(
            url.includes(secondCorpusId)
              ? corpusWithSnapshot(secondSnapshot, secondCorpusId)
              : corpusWithSnapshot(firstSnapshot),
          ),
        );
      });
    const user = userEvent.setup();
    const provider = createHttpResearchProvider({ fetch: fetchResponse });

    renderAtRoute(
      <Routes>
        <Route
          path="corpora/:corpusId"
          element={<WorkspaceRouteChange provider={provider} />}
        />
      </Routes>,
      `/corpora/${firstCorpusId}`,
    );

    await waitFor(() => expect(firstSignal).toBeInstanceOf(AbortSignal));
    await user.click(screen.getByRole("button", { name: "Change corpus" }));

    expect(firstSignal?.aborted).toBe(true);
    expect(
      await screen.findByRole("button", { name: "Second corpus question" }),
    ).toBeVisible();
    firstResponse.resolve(
      jsonResponse(
        openingSuggestionsResponse(firstCorpusId, "en", firstSnapshot, [
          {
            caseId: "first-corpus-question",
            rank: 1,
            question: "First corpus question",
          },
        ]),
      ),
    );
    await waitFor(() =>
      expect(
        screen.queryByText("First corpus question"),
      ).not.toBeInTheDocument(),
    );
  });

  it("aborts and discards a pending projection when publication refreshes the active snapshot", async () => {
    const firstSnapshot = activeSnapshot();
    const secondSnapshot = activeSnapshot(
      "70000000-0000-4000-8000-000000000002",
      "f".repeat(64),
    );
    const firstResponse = deferred<Response>();
    let currentSnapshot = firstSnapshot;
    let firstSignal: AbortSignal | undefined;
    let suggestionRequestCount = 0;
    const staleProjectionProcessed = deferred<undefined>();
    const fetchResponse = vi
      .fn<typeof fetch>()
      .mockImplementation((input, init) => {
        const url = requestUrl(input);
        if (url.includes("/opening-suggestions?")) {
          suggestionRequestCount += 1;
          if (suggestionRequestCount === 1) {
            firstSignal = init?.signal ?? undefined;
            return firstResponse.promise;
          }
          return Promise.resolve(
            jsonResponse(
              openingSuggestionsResponse(corpus().id, "en", secondSnapshot, [
                {
                  caseId: "new-snapshot-question",
                  rank: 1,
                  question: "New snapshot question",
                },
              ]),
            ),
          );
        }
        if (url.endsWith("/snapshots") && init?.method === "POST") {
          currentSnapshot = secondSnapshot;
          return Promise.resolve(
            jsonResponse(snapshotPublicationResponse(secondSnapshot)),
          );
        }
        if (url.endsWith("/sources"))
          return Promise.resolve(jsonResponse([source()]));
        if (url.includes("/document"))
          return Promise.resolve(jsonResponse(document()));
        return Promise.resolve(
          jsonResponse(corpusWithSnapshot(currentSnapshot)),
        );
      });
    const provider = createHttpResearchProvider({ fetch: fetchResponse });
    const getCorpusOpeningSuggestions =
      provider.getCorpusOpeningSuggestions.bind(provider);
    vi.spyOn(provider, "getCorpusOpeningSuggestions").mockImplementation(
      async (...arguments_) => {
        try {
          return await getCorpusOpeningSuggestions(...arguments_);
        } finally {
          if (arguments_[2] === firstSignal)
            staleProjectionProcessed.resolve(undefined);
        }
      },
    );
    const user = userEvent.setup();

    renderAtRoute(
      <Routes>
        <Route
          path="corpora/:corpusId"
          element={<CorpusWorkspacePage provider={provider} />}
        />
      </Routes>,
      `/corpora/${corpus().id}`,
    );

    await waitFor(() => expect(firstSignal).toBeInstanceOf(AbortSignal));
    await user.click(screen.getByRole("tab", { name: "Source" }));
    await user.click(
      screen.getByRole("treeitem", {
        name: "Official English GDPR text (Ready)",
      }),
    );
    await user.click(
      await screen.findByRole("button", { name: "Publish snapshot" }),
    );

    await waitFor(() => expect(firstSignal?.aborted).toBe(true));
    await user.click(screen.getByRole("tab", { name: "Chat" }));
    expect(
      await screen.findByRole("button", { name: "New snapshot question" }),
    ).toBeVisible();
    expect(suggestionRequestCount).toBe(2);
    await act(async () => {
      firstResponse.resolve(
        jsonResponse(
          openingSuggestionsResponse(corpus().id, "en", firstSnapshot, [
            {
              caseId: "old-snapshot-question",
              rank: 1,
              question: "Old snapshot question",
            },
          ]),
        ),
      );
      await staleProjectionProcessed.promise;
      await Promise.resolve();
    });

    expect(screen.queryByText("Old snapshot question")).not.toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "New snapshot question" }),
    ).toBeVisible();
  });
});

function renderWorkspace(
  fetchResponse: typeof fetch,
  chatProvider?: ChatProvider,
) {
  renderAtRoute(
    <Routes>
      <Route
        path="corpora/:corpusId"
        element={
          <CorpusWorkspacePage
            provider={createHttpResearchProvider({ fetch: fetchResponse })}
            chatProvider={chatProvider}
          />
        }
      />
    </Routes>,
    "/corpora/10000000-0000-4000-8000-000000000002",
  );
}

function corpus(): CorpusResponse {
  return {
    id: "10000000-0000-4000-8000-000000000002",
    name: "EU Privacy Law",
    description: "Official materials.",
    language: "en",
    jurisdiction: "European Union",
    status: "enabled",
    sourceCount: 1,
    version: 1,
    createdAt: "2026-08-17T12:00:00Z",
    updatedAt: "2026-08-17T12:00:00Z",
  };
}

function corpusWithSnapshot(
  activeSnapshot: ActiveSnapshotResponse,
  id = corpus().id,
): CorpusResponse {
  return { ...corpus(), id, activeSnapshot };
}

function activeSnapshot(
  id = "70000000-0000-4000-8000-000000000001",
  manifestSha256 = "e".repeat(64),
): ActiveSnapshotResponse {
  return {
    id,
    manifestSha256,
    createdAt: "2026-08-17T12:00:00Z",
    activatedAt: "2026-08-17T12:00:00Z",
    releaseVersion: id.endsWith("2") ? 2 : 1,
  };
}

function snapshotPublicationResponse(release: ActiveSnapshotResponse) {
  return {
    snapshot: {
      id: release.id,
      corpusId: corpus().id,
      manifestSha256: release.manifestSha256,
      createdBy: "local-maintainer",
      createdAt: release.createdAt,
      members: [],
    },
    release,
    published: true,
  };
}

function openingSuggestionsResponse(
  corpusId: string,
  interfaceLanguage: "en" | "pt",
  snapshot: ActiveSnapshotResponse,
  suggestions: readonly {
    readonly caseId: string;
    readonly rank: number;
    readonly question: string;
  }[],
) {
  return {
    corpusId,
    activeSnapshotId: snapshot.id,
    activeSnapshotManifestSha256: snapshot.manifestSha256,
    interfaceLanguage,
    suggestions,
  };
}

function deferred<Value>() {
  let resolve: (value: Value) => void;
  const promise = new Promise<Value>((promiseResolve) => {
    resolve = promiseResolve;
  });
  return { promise, resolve: (value: Value) => resolve(value) };
}

function WorkspaceRouteChange({
  provider,
}: {
  readonly provider: ResearchProvider;
}) {
  const navigate = useNavigate();
  return (
    <>
      <button
        type="button"
        onClick={() => {
          void navigate("/corpora/10000000-0000-4000-8000-000000000003");
        }}
      >
        Change corpus
      </button>
      <CorpusWorkspacePage provider={provider} />
    </>
  );
}

function source() {
  return {
    id: "20000000-0000-4000-8000-000000000002",
    corpusId: "10000000-0000-4000-8000-000000000002",
    title: "Official English GDPR text",
    kind: "url",
    processingStatus: "ready",
    failureCategory: null,
    latestReadyDocumentId: "50000000-0000-4000-8000-000000000001",
    version: 2,
    createdAt: "2026-08-17T12:00:00Z",
    updatedAt: "2026-08-17T12:01:00Z",
    origin: {
      kind: "url" as const,
      submittedUrl: "https://example.org/gdpr",
      normalizedUrl: "https://example.org/gdpr",
      originalFilename: null,
      mediaType: "text/html",
      byteSize: 2048,
      sha256: "b".repeat(64),
      finalUrl: "https://example.org/gdpr",
      capturedAt: "2026-08-17T12:00:30Z",
      extractedContentSha256: "a".repeat(64),
    },
    latestAttempt: null,
    attempts: [],
  };
}

function document() {
  return {
    id: "50000000-0000-4000-8000-000000000001",
    sourceRevisionId: "40000000-0000-4000-8000-000000000001",
    pipelineVersion: "corpus-ingestion-v1",
    text: "Persisted legal text.",
    textSha256: "a".repeat(64),
    createdAt: "2026-08-17T12:01:00Z",
    provenance: {
      contentSha256: "b".repeat(64),
      capturedAt: "2026-08-17T12:00:30Z",
      mediaType: "text/html",
      byteSize: 2048,
      finalUrl: "https://example.org/gdpr",
      extractedContentSha256: "a".repeat(64),
    },
    units: [
      {
        id: "60000000-0000-4000-8000-000000000000",
        parentId: null,
        kind: "document",
        ordinal: 0,
        marker: null,
        label: null,
        startOffset: 0,
        endOffset: 21,
        startPage: null,
        endPage: null,
        locator: "document",
        contentSha256: "c".repeat(64),
      },
      {
        id: "60000000-0000-4000-8000-000000000001",
        parentId: "60000000-0000-4000-8000-000000000000",
        kind: "article",
        ordinal: 0,
        marker: "Article 1",
        label: "Article 1",
        startOffset: 0,
        endOffset: 21,
        startPage: null,
        endPage: null,
        locator: "article-1",
        contentSha256: "d".repeat(64),
      },
    ],
  };
}

function jsonResponse(value: unknown): Response {
  return new Response(JSON.stringify(value), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

function requestUrl(input: RequestInfo | URL): string {
  return typeof input === "string"
    ? input
    : input instanceof URL
      ? input.href
      : input.url;
}
