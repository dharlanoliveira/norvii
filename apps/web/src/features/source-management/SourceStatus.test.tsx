import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { SnapshotResponse, SourceResponse } from "../../api/contract";
import { ResearchRequestError } from "../../api/researchProvider";
import { renderAtRoute } from "../../test/render";
import { SourceStatus } from "./SourceStatus";

describe("source lifecycle status", () => {
  it("offers retry for a failed source with a safe failure diagnostic", async () => {
    const user = userEvent.setup();
    const retry = vi.fn().mockResolvedValue(undefined);
    renderAtRoute(
      <SourceStatus
        source={source("failed")}
        onLoadGraphRelease={vi.fn().mockRejectedValue(new Error("not used"))}
        onLoadSnapshotHistory={vi.fn().mockResolvedValue([])}
        onPublish={vi.fn()}
        onRetry={retry}
        onReprocess={vi.fn()}
      />,
    );

    expect(
      screen.getByText(
        "The latest attempt ended with a safe failure category.",
      ),
    ).toBeVisible();
    await user.click(screen.getByText("Source details"));
    expect(screen.getByText("https://example.org/law")).toBeVisible();
    expect(screen.getAllByText("corpus-ingestion-v1")).toHaveLength(3);
    expect(screen.getByText("provider_response_invalid")).toBeVisible();
    expect(screen.getByRole("heading", { name: "Attempt 4" })).toBeVisible();
    expect(screen.getByRole("heading", { name: "Attempt 2" })).toBeVisible();
    expect(
      screen.queryByRole("heading", { name: "Attempt 1" }),
    ).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Retry ingestion" }));
    expect(retry).toHaveBeenCalledWith(
      expect.objectContaining({ version: 3 }),
      expect.any(AbortSignal),
    );
  });

  it("offers explicit reprocessing only for a ready source", async () => {
    const user = userEvent.setup();
    const reprocess = vi.fn().mockResolvedValue(undefined);
    const confirm = vi.spyOn(globalThis, "confirm").mockReturnValue(true);
    renderAtRoute(
      <SourceStatus
        source={source("ready")}
        onLoadGraphRelease={vi.fn().mockRejectedValue(new Error("not used"))}
        onLoadSnapshotHistory={vi.fn().mockResolvedValue([])}
        onPublish={vi.fn()}
        onRetry={vi.fn()}
        onReprocess={reprocess}
      />,
    );

    expect(
      screen.getByRole("heading", { name: "Official source" }),
    ).toBeVisible();
    const actions = screen.getByRole("toolbar", { name: "Source actions" });
    expect(within(actions).getByText("Source details")).toBeVisible();
    expect(
      within(actions).getByRole("button", { name: "Reprocess source" }),
    ).toBeVisible();
    expect(
      within(actions).getByRole("link", { name: "Open official source" }),
    ).toHaveAttribute(
      "href",
      "/api/v1/corpora/10000000-0000-4000-8000-000000000009/sources/20000000-0000-4000-8000-000000000009/origin",
    );
    await user.click(screen.getByRole("button", { name: "Reprocess source" }));
    expect(reprocess).toHaveBeenCalledOnce();
    expect(confirm).toHaveBeenCalledWith(
      expect.stringContaining(
        "The current ready document will remain available",
      ),
    );
    expect(
      screen.queryByRole("button", { name: "Retry ingestion" }),
    ).not.toBeInTheDocument();
  });

  it("publishes a ready document through the explicit snapshot action", async () => {
    const user = userEvent.setup();
    const publish = vi.fn().mockResolvedValue(undefined);
    renderAtRoute(
      <SourceStatus
        source={source("ready")}
        activeSnapshot={{
          id: "70000000-0000-4000-8000-000000000001",
          manifestSha256: "a".repeat(64),
          createdAt: "2026-08-24T12:00:00Z",
          activatedAt: "2026-08-24T12:00:00Z",
          releaseVersion: 1,
        }}
        onLoadGraphRelease={vi.fn().mockRejectedValue(new Error("not used"))}
        onLoadSnapshotHistory={vi.fn().mockResolvedValue([])}
        onPublish={publish}
        onRetry={vi.fn()}
        onReprocess={vi.fn()}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Publish snapshot" }));

    expect(publish).toHaveBeenCalledWith(
      source("ready"),
      expect.any(AbortSignal),
    );
  });

  it("does not reprocess when confirmation is declined", async () => {
    const user = userEvent.setup();
    const reprocess = vi.fn().mockResolvedValue(undefined);
    vi.spyOn(globalThis, "confirm").mockReturnValue(false);
    renderAtRoute(
      <SourceStatus
        source={source("ready")}
        onLoadGraphRelease={vi.fn().mockRejectedValue(new Error("not used"))}
        onLoadSnapshotHistory={vi.fn().mockResolvedValue([])}
        onPublish={vi.fn()}
        onRetry={vi.fn()}
        onReprocess={reprocess}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Reprocess source" }));

    expect(reprocess).not.toHaveBeenCalled();
  });

  it("loads immutable release manifests only when requested", async () => {
    const user = userEvent.setup();
    const snapshots: readonly SnapshotResponse[] = [
      {
        id: "70000000-0000-4000-8000-000000000001",
        corpusId: "10000000-0000-4000-8000-000000000009",
        manifestSha256: "a".repeat(64),
        createdBy: "local-maintainer",
        createdAt: "2026-08-24T12:00:00Z",
        members: [
          {
            sourceId: "20000000-0000-4000-8000-000000000009",
            sourceRevisionId: "30000000-0000-4000-8000-000000000009",
            documentId: "50000000-0000-4000-8000-000000000009",
            officialOrigin: "https://example.org/law",
            capturedAt: "2026-08-24T11:00:00Z",
            contentSha256: "b".repeat(64),
          },
        ],
      },
    ];
    const loadSnapshots = vi.fn(
      (signal: AbortSignal): Promise<readonly SnapshotResponse[]> => {
        void signal;
        return Promise.resolve(snapshots);
      },
    );
    renderAtRoute(
      <SourceStatus
        source={source("ready")}
        activeSnapshot={{
          id: "70000000-0000-4000-8000-000000000001",
          manifestSha256: "a".repeat(64),
          createdAt: "2026-08-24T12:00:00Z",
          activatedAt: "2026-08-24T12:00:00Z",
          releaseVersion: 1,
        }}
        onLoadGraphRelease={vi.fn().mockResolvedValue({
          id: "80000000-0000-4000-8000-000000000001",
          corpusId: "10000000-0000-4000-8000-000000000009",
          snapshotId: "70000000-0000-4000-8000-000000000001",
          manifestSha256: "c".repeat(64),
          buildVersion: "legal-graph-v1",
          status: "ready",
          failureCategory: null,
          entityCount: 4,
          relationshipCount: 2,
          createdAt: "2026-08-24T12:00:00Z",
          completedAt: "2026-08-24T12:01:00Z",
        })}
        onLoadSnapshotHistory={loadSnapshots}
        onPublish={vi.fn()}
        onRetry={vi.fn()}
        onReprocess={vi.fn()}
      />,
    );

    expect(loadSnapshots).not.toHaveBeenCalled();
    await user.click(screen.getByText("Snapshot history"));
    expect(await screen.findByText("Snapshot release")).toBeVisible();
    expect(screen.getByText("Active release")).toBeVisible();
    expect(screen.getByText("Graph release")).toBeVisible();
    expect(loadSnapshots).toHaveBeenCalledWith(expect.any(AbortSignal));
  });

  it("explains a failed lifecycle action without losing the source state", async () => {
    const user = userEvent.setup();
    renderAtRoute(
      <SourceStatus
        source={source("failed")}
        onLoadGraphRelease={vi.fn().mockRejectedValue(new Error("not used"))}
        onLoadSnapshotHistory={vi.fn().mockResolvedValue([])}
        onPublish={vi.fn()}
        onRetry={vi.fn().mockRejectedValue(new Error("unavailable"))}
        onReprocess={vi.fn()}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Retry ingestion" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "The source action could not be queued.",
    );
    expect(
      screen.getByRole("button", { name: "Retry ingestion" }),
    ).toBeEnabled();
  });

  it("asks the researcher to reload after a stale publication attempt", async () => {
    const user = userEvent.setup();
    renderAtRoute(
      <SourceStatus
        source={source("ready")}
        activeSnapshot={activeSnapshot()}
        onLoadGraphRelease={vi.fn().mockRejectedValue(new Error("not used"))}
        onLoadSnapshotHistory={vi.fn().mockResolvedValue([])}
        onPublish={vi
          .fn()
          .mockRejectedValue(
            new ResearchRequestError(
              "stale_state",
              "stale",
              undefined,
              "request-1",
            ),
          )}
        onRetry={vi.fn()}
        onReprocess={vi.fn()}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Publish snapshot" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "This source changed. Reload before trying the action again.",
    );
  });

  it("offers the preserved PDF download when the source is a PDF", () => {
    const readySource = source("ready");
    renderAtRoute(
      <SourceStatus
        source={{
          ...readySource,
          kind: "pdf",
          origin: {
            ...readySource.origin,
            kind: "pdf",
            submittedUrl: null,
            normalizedUrl: null,
            originalFilename: "law.pdf",
            mediaType: "application/pdf",
          },
        }}
        onLoadGraphRelease={vi.fn().mockRejectedValue(new Error("not used"))}
        onLoadSnapshotHistory={vi.fn().mockResolvedValue([])}
        onPublish={vi.fn()}
        onRetry={vi.fn()}
        onReprocess={vi.fn()}
      />,
    );

    expect(
      screen.getByRole("link", { name: "Download preserved PDF" }),
    ).toHaveAttribute(
      "href",
      "/api/v1/corpora/10000000-0000-4000-8000-000000000009/sources/20000000-0000-4000-8000-000000000009/origin/pdf",
    );
  });
});

function activeSnapshot() {
  return {
    id: "70000000-0000-4000-8000-000000000001",
    manifestSha256: "a".repeat(64),
    createdAt: "2026-08-24T12:00:00Z",
    activatedAt: "2026-08-24T12:00:00Z",
    releaseVersion: 1,
  };
}

function source(
  processingStatus: SourceResponse["processingStatus"],
): SourceResponse {
  return {
    id: "20000000-0000-4000-8000-000000000009",
    corpusId: "10000000-0000-4000-8000-000000000009",
    title: "Official source",
    kind: "url",
    processingStatus,
    failureCategory:
      processingStatus === "failed" ? "acquisition_failed" : null,
    latestReadyDocumentId:
      processingStatus === "ready"
        ? "50000000-0000-4000-8000-000000000009"
        : null,
    activeSnapshotDocumentId: null,
    version: 3,
    createdAt: "2026-08-17T12:00:00Z",
    updatedAt: "2026-08-17T12:01:00Z",
    origin: {
      kind: "url",
      submittedUrl: "https://example.org/law",
      normalizedUrl: "https://example.org/law",
      originalFilename: null,
      mediaType: "text/html",
      byteSize: 1200,
      sha256: "a".repeat(64),
      finalUrl: "https://example.org/law",
      capturedAt: "2026-08-17T12:00:30Z",
      extractedContentSha256: "b".repeat(64),
    },
    latestAttempt: {
      number: 4,
      pipelineVersion: "corpus-ingestion-v1",
      status: processingStatus === "failed" ? "failed" : "succeeded",
      startedAt: "2026-08-17T12:00:00Z",
      finishedAt: "2026-08-17T12:01:00Z",
      failureCategory:
        processingStatus === "failed" ? "acquisition_failed" : null,
      failureDetail:
        processingStatus === "failed" ? "provider_response_invalid" : null,
      acquiredByteCount: 1200,
      normalizedCharacterCount: 800,
      unitCount: 4,
      durationMilliseconds: 1000,
    },
    attempts: [
      {
        number: 4,
        pipelineVersion: "corpus-ingestion-v1",
        status: processingStatus === "failed" ? "failed" : "succeeded",
        startedAt: "2026-08-17T12:00:00Z",
        finishedAt: "2026-08-17T12:01:00Z",
        failureCategory:
          processingStatus === "failed" ? "acquisition_failed" : null,
        failureDetail:
          processingStatus === "failed" ? "provider_response_invalid" : null,
        acquiredByteCount: 1200,
        normalizedCharacterCount: 800,
        unitCount: 4,
        durationMilliseconds: 1000,
      },
      {
        number: 3,
        pipelineVersion: "corpus-ingestion-v1",
        status: "failed",
        startedAt: "2026-08-17T11:30:00Z",
        finishedAt: "2026-08-17T11:31:00Z",
        failureCategory: "acquisition_failed",
        failureDetail: null,
        acquiredByteCount: null,
        normalizedCharacterCount: null,
        unitCount: null,
        durationMilliseconds: 1000,
      },
      {
        number: 2,
        pipelineVersion: "corpus-ingestion-v1",
        status: "failed",
        startedAt: "2026-08-17T11:15:00Z",
        finishedAt: "2026-08-17T11:16:00Z",
        failureCategory: "acquisition_failed",
        failureDetail: null,
        acquiredByteCount: null,
        normalizedCharacterCount: null,
        unitCount: null,
        durationMilliseconds: 1000,
      },
      {
        number: 1,
        pipelineVersion: "corpus-ingestion-v1",
        status: "failed",
        startedAt: "2026-08-17T11:00:00Z",
        finishedAt: "2026-08-17T11:01:00Z",
        failureCategory: "acquisition_failed",
        failureDetail: null,
        acquiredByteCount: null,
        normalizedCharacterCount: null,
        unitCount: null,
        durationMilliseconds: 1000,
      },
    ],
  };
}
