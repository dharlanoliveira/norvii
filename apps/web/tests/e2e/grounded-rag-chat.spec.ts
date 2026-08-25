import { expect, test, type Page } from "@playwright/test";

const corpusId = "10000000-0000-4000-8000-000000000005";
const foreignCorpusId = "10000000-0000-4000-8000-000000000006";
const snapshotId = "30000000-0000-4000-8000-000000000005";
const sourceId = "20000000-0000-4000-8000-000000000005";
const documentId = "50000000-0000-4000-8000-000000000005";

test("uses the active corpus snapshot and opens its immutable citation", async ({
  page,
}) => {
  let chatRequest: unknown;
  await configureWorkspace(page, completedStream(reference()));
  await page.route(
    `**/api/v1/corpora/${corpusId}/chat/stream`,
    async (route) => {
      chatRequest = route.request().postDataJSON();
      await route.fulfill({
        body: completedStream(reference()),
        contentType: "text/event-stream",
      });
    },
  );

  await ask(page, "What does Article 1 protect?");

  await expect(
    page.getByText("Article 1 protects the active snapshot interest."),
  ).toBeVisible();
  expect(chatRequest).toEqual({
    question: "What does Article 1 protect?",
    interfaceLanguage: "en",
    strategy: "hybrid",
  });

  await page.locator("details.answer-inspection > summary").click();
  await expect(page.getByText(snapshotId)).toBeVisible();
  await page
    .getByRole("button", { name: "Open Article 1 in Active GDPR text" })
    .click();
  await expect(page.locator("mark.legal-citation-highlight")).toHaveText(
    "Active snapshot legal text.",
  );
});

test("fails closed when a chat service returns a foreign-corpus citation", async ({
  page,
}) => {
  let immutableDocumentRequests = 0;
  const foreignReference = {
    ...reference(),
    corpusId: foreignCorpusId,
    snapshotId: "30000000-0000-4000-8000-000000000006",
    sourceTitle: "Foreign corpus source",
  };
  await configureWorkspace(page, completedStream(foreignReference));
  await page.route(
    `**/api/v1/corpora/${corpusId}/sources/${sourceId}/documents/${documentId}`,
    async (route) => {
      immutableDocumentRequests += 1;
      await route.fulfill({ json: immutableDocument() });
    },
  );

  await ask(page, "What does Article 1 protect?");
  await page
    .getByRole("button", {
      name: "Open Article 1 in Foreign corpus source",
    })
    .click();
  await page.getByRole("tab", { name: "Source" }).click();

  await expect(page.getByRole("alert")).toHaveText(
    "The cited location is unavailable.",
  );
  expect(immutableDocumentRequests).toBe(0);
});

test("renders the service abstention without citations or a completed answer", async ({
  page,
}) => {
  await configureWorkspace(
    page,
    stream([
      started(),
      {
        type: "abstained",
        requestId: "70000000-0000-4000-8000-000000000005",
        reason: "insufficient_evidence",
        telemetry: telemetry("abstained", 0),
      },
    ]),
  );

  await ask(page, "What is outside the active evidence?");

  await expect(
    page.getByText(
      "I could not find enough published evidence in this corpus to answer safely.",
    ),
  ).toBeVisible();
  await expect(page.getByRole("button", { name: /Open Article/ })).toHaveCount(
    0,
  );
});

test("renders cancellation as the only terminal outcome after streamed text", async ({
  page,
}) => {
  await configureWorkspace(
    page,
    stream([
      started(),
      {
        type: "evidence",
        requestId: "70000000-0000-4000-8000-000000000005",
        references: [reference()],
      },
      {
        type: "delta",
        requestId: "70000000-0000-4000-8000-000000000005",
        text: "Partial response.",
      },
      {
        type: "cancelled",
        requestId: "70000000-0000-4000-8000-000000000005",
        telemetry: telemetry("cancelled", 1),
      },
    ]),
  );

  await ask(page, "What does Article 1 protect?");

  await expect(page.getByText("Response cancelled.")).toBeVisible();
  await expect(page.getByText("Partial response.")).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Research record" }),
  ).toHaveCount(0);
});

async function ask(page: Page, question: string): Promise<void> {
  await page.goto(`/corpora/${corpusId}`);
  await page.getByRole("textbox", { name: "Research question" }).fill(question);
  await page.getByRole("button", { name: "Send question" }).click();
}

async function configureWorkspace(
  page: Page,
  chatStream: string,
): Promise<void> {
  await page.route(`**/api/v1/corpora/${corpusId}`, async (route) =>
    route.fulfill({ json: corpus() }),
  );
  await page.route(`**/api/v1/corpora/${corpusId}/sources`, async (route) =>
    route.fulfill({ json: [source()] }),
  );
  await page.route(
    `**/api/v1/corpora/${corpusId}/sources/${sourceId}/document`,
    async (route) => route.fulfill({ json: currentDocument() }),
  );
  await page.route(
    `**/api/v1/corpora/${corpusId}/sources/${sourceId}/documents/${documentId}`,
    async (route) => route.fulfill({ json: immutableDocument() }),
  );
  await page.route(`**/api/v1/corpora/${corpusId}/chat/stream`, async (route) =>
    route.fulfill({ body: chatStream, contentType: "text/event-stream" }),
  );
}

function corpus() {
  return {
    id: corpusId,
    name: "Active GDPR corpus",
    description: "Active corpus test fixture.",
    language: "en",
    jurisdiction: "Test",
    status: "enabled",
    sourceCount: 1,
    version: 1,
    createdAt: "2026-08-25T12:00:00Z",
    updatedAt: "2026-08-25T12:00:00Z",
    activeSnapshot: {
      id: snapshotId,
      manifestSha256: "a".repeat(64),
      createdAt: "2026-08-25T12:00:00Z",
      activatedAt: "2026-08-25T12:00:00Z",
      releaseVersion: 1,
    },
  };
}

function source() {
  return {
    id: sourceId,
    corpusId,
    title: "Active GDPR text",
    kind: "url",
    processingStatus: "ready",
    failureCategory: null,
    latestReadyDocumentId: documentId,
    activeSnapshotDocumentId: documentId,
    version: 1,
    createdAt: "2026-08-25T12:00:00Z",
    updatedAt: "2026-08-25T12:00:00Z",
    origin: {
      kind: "url",
      submittedUrl: "https://example.org/active-gdpr",
      normalizedUrl: "https://example.org/active-gdpr",
      originalFilename: null,
      mediaType: "text/html",
      byteSize: 100,
      sha256: "b".repeat(64),
      finalUrl: "https://example.org/active-gdpr",
      capturedAt: "2026-08-25T12:00:00Z",
      extractedContentSha256: "c".repeat(64),
    },
    latestAttempt: null,
    attempts: [],
  };
}

function currentDocument() {
  return document("Current Article 1 legal text.");
}

function immutableDocument() {
  return document("Active snapshot legal text.");
}

function document(text: string) {
  const article = `Article 1\n${text}`;
  return {
    id: documentId,
    sourceRevisionId: "40000000-0000-4000-8000-000000000005",
    pipelineVersion: "grounded-chat-e2e",
    text: article,
    textSha256: "c".repeat(64),
    createdAt: "2026-08-25T12:00:00Z",
    provenance: {
      contentSha256: "b".repeat(64),
      capturedAt: "2026-08-25T12:00:00Z",
      mediaType: "text/html",
      byteSize: 100,
      finalUrl: "https://example.org/active-gdpr",
      extractedContentSha256: "c".repeat(64),
    },
    units: [
      {
        id: "60000000-0000-4000-8000-000000000005",
        parentId: null,
        kind: "document",
        ordinal: 0,
        marker: null,
        label: null,
        startOffset: 0,
        endOffset: article.length,
        startPage: null,
        endPage: null,
        locator: "document",
        contentSha256: "d".repeat(64),
      },
      {
        id: "60000000-0000-4000-8000-000000000006",
        parentId: "60000000-0000-4000-8000-000000000005",
        kind: "article",
        ordinal: 1,
        marker: "Article 1",
        label: "Article 1",
        startOffset: 0,
        endOffset: article.length,
        startPage: null,
        endPage: null,
        locator: "article-1",
        contentSha256: "e".repeat(64),
      },
    ],
  };
}

function reference() {
  return {
    id: "reference-1",
    corpusId,
    snapshotId,
    sourceId,
    documentId,
    documentVersionId: documentId,
    sourceRevisionId: "40000000-0000-4000-8000-000000000005",
    pipelineVersion: "grounded-chat-e2e",
    sourceTitle: "Active GDPR text",
    unitLocator: "article-1",
    startOffset: 10,
    endOffset: 37,
    excerpt: "Active snapshot legal text.",
    rank: 1,
    cosineDistance: 0.1,
    contribution: "vector",
  };
}

function started() {
  return {
    type: "started",
    requestId: "70000000-0000-4000-8000-000000000005",
    corpusId,
  };
}

function completedStream(referenceValue: ReturnType<typeof reference>): string {
  return stream([
    started(),
    {
      type: "evidence",
      requestId: "70000000-0000-4000-8000-000000000005",
      references: [referenceValue],
    },
    {
      type: "delta",
      requestId: "70000000-0000-4000-8000-000000000005",
      text: "Article 1 protects the active snapshot interest.",
    },
    {
      type: "completed",
      requestId: "70000000-0000-4000-8000-000000000005",
      answer: "Article 1 protects the active snapshot interest.",
      references: [referenceValue],
      telemetry: telemetry("completed", 1),
      inspection: {
        outcome: "completed",
        retrieval: {
          strategy: "hybrid",
          topK: 8,
          returnedCount: 1,
          embeddingModel: "deterministic-e2e",
        },
        measurements: {
          retrievalMilliseconds: 1,
          generationMilliseconds: 1,
          totalMilliseconds: 2,
          inputTokens: null,
          outputTokens: null,
        },
        evidence: [referenceValue],
        graphPath: [],
        stages: [],
      },
    },
  ]);
}

function telemetry(outcome: string, evidenceCount: number) {
  return { outcome, evidenceCount, durationMilliseconds: 2 };
}

function stream(events: readonly object[]): string {
  return events.map((event) => `data: ${JSON.stringify(event)}\n\n`).join("");
}
