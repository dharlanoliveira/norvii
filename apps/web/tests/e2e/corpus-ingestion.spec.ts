import { expect, test, type Page } from "@playwright/test";

const corpusId = "10000000-0000-4000-8000-000000000002";
const sourceId = "20000000-0000-4000-8000-000000000002";

test.beforeEach(async ({ page }) => configureAuthoritativeAPI(page));

test("browses a persisted document by keyboard and opens grounded chat", async ({
  page,
}) => {
  await page.goto("/");
  await expect(page.locator("html")).toHaveAttribute("lang", "en");
  await page.getByRole("link", { name: /Open corpus EU Privacy Law/ }).click();

  const source = page.getByRole("treeitem", { name: /Official GDPR text/ });
  await source.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByText("Persisted Article 1 legal text.")).toBeVisible();
  await expect(
    page.getByRole("navigation", { name: "Document locations" }),
  ).toBeVisible();
  const location = page.getByRole("combobox", { name: "Document location" });
  await location.focus();
  await page.keyboard.press("ArrowDown");
  await expect(
    page.getByRole("article", { name: "Article 2", exact: true }),
  ).toHaveAttribute("data-selected", "true");
  await expect(location).toHaveValue("60000000-0000-4000-8000-000000000002");
  await expect(
    page.getByRole("heading", { name: "Ask about this corpus." }),
  ).toBeHidden();

  await page.getByRole("tab", { name: "Chat" }).click();
  await expect(
    page.getByRole("heading", { name: "Ask about this corpus." }),
  ).toBeVisible();
  await expect(
    page.getByRole("textbox", { name: "Research question" }),
  ).toBeVisible();

  await page
    .getByRole("combobox", { name: "Interface language" })
    .selectOption("pt");
  await expect(
    page.getByRole("heading", {
      name: "Pergunte sobre este corpus.",
    }),
  ).toBeVisible();
  await expect(page).toHaveURL(new RegExp(`/corpora/${corpusId}$`));
});

test("shows authoritative empty and failed outcomes", async ({ page }) => {
  await page.route("**/api/v1/corpora?includeDisabled=true", async (route) =>
    route.fulfill({ json: [] }),
  );
  await page.goto("/");
  await expect(page.getByText("No corpora are available yet.")).toBeVisible();

  await page.route("**/api/v1/corpora?includeDisabled=true", async (route) =>
    route.fulfill({ status: 503, json: errorEnvelope("unavailable") }),
  );
  await page.reload();
  await expect(page.getByRole("alert")).toBeVisible();
  await expect(page.getByRole("article")).toHaveCount(0);
});

async function configureAuthoritativeAPI(page: Page): Promise<void> {
  await page.route("**/api/v1/corpora?includeDisabled=true", async (route) =>
    route.fulfill({ json: [corpus()] }),
  );
  await page.route(`**/api/v1/corpora/${corpusId}`, async (route) =>
    route.fulfill({ json: corpus() }),
  );
  await page.route(`**/api/v1/corpora/${corpusId}/sources`, async (route) =>
    route.fulfill({ json: [source()] }),
  );
  await page.route(
    `**/api/v1/corpora/${corpusId}/sources/${sourceId}/document`,
    async (route) => route.fulfill({ json: documentResponse() }),
  );
}

function corpus() {
  return {
    id: corpusId,
    name: "EU Privacy Law",
    description: "Official EU privacy materials.",
    language: "en",
    jurisdiction: "European Union",
    status: "enabled",
    sourceCount: 1,
    version: 1,
    createdAt: "2026-08-17T12:00:00Z",
    updatedAt: "2026-08-17T12:00:00Z",
  };
}

function source() {
  return {
    id: sourceId,
    corpusId,
    title: "Official GDPR text",
    kind: "url",
    processingStatus: "ready",
    failureCategory: null,
    latestReadyDocumentId: "50000000-0000-4000-8000-000000000001",
    version: 3,
    createdAt: "2026-08-17T12:00:00Z",
    updatedAt: "2026-08-17T12:01:00Z",
    origin: {
      kind: "url",
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

function documentResponse() {
  const articleOne = "Article 1\nPersisted Article 1 legal text.";
  const articleTwo = "Article 2\nPersisted Article 2 legal text.";
  const text = `${articleOne}\n${articleTwo}`;
  return {
    id: "50000000-0000-4000-8000-000000000001",
    sourceRevisionId: "40000000-0000-4000-8000-000000000001",
    pipelineVersion: "corpus-ingestion-v1",
    text,
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
        endOffset: text.length,
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
        endOffset: articleOne.length,
        startPage: null,
        endPage: null,
        locator: "article-1",
        contentSha256: "b".repeat(64),
      },
      {
        id: "60000000-0000-4000-8000-000000000002",
        parentId: "60000000-0000-4000-8000-000000000000",
        kind: "article",
        ordinal: 1,
        marker: "Article 2",
        label: "Article 2",
        startOffset: articleOne.length + 1,
        endOffset: text.length,
        startPage: null,
        endPage: null,
        locator: "article-2",
        contentSha256: "d".repeat(64),
      },
    ],
  };
}

function errorEnvelope(code: string) {
  return {
    error: {
      code,
      message: "The service is unavailable.",
      requestId: "90000000-0000-4000-8000-000000000001",
    },
  };
}
