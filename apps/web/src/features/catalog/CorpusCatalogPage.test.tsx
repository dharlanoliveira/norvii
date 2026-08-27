import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import {
  createHttpResearchProvider,
  ResearchRequestError,
} from "../../api/researchProvider";
import type { CorpusResponse } from "../../api/contract";
import type {
  CorpusDraft,
  CorpusUpdate,
  ResearchProvider,
} from "../../research/domain/authoritative";
import {
  expectNoAccessibilityViolations,
  renderAtRoute,
} from "../../test/render";
import { CorpusCatalogPage } from "./CorpusCatalogPage";

describe("authoritative corpus catalog", () => {
  it("renders real API data after an accessible loading state", async () => {
    const fetchResponse = vi
      .fn<typeof fetch>()
      .mockResolvedValue(
        jsonResponse([
          corpus(
            "10000000-0000-4000-8000-000000000002",
            "EU Privacy Law",
            "en",
          ),
          corpus(
            "10000000-0000-4000-8000-000000000001",
            "Brazil Privacy Law",
            "pt",
          ),
        ]),
      );
    const result = renderAtRoute(
      <CorpusCatalogPage
        provider={createHttpResearchProvider({ fetch: fetchResponse })}
      />,
    );

    expect(screen.getByRole("status")).toBeVisible();
    expect(
      await screen.findByRole("heading", { name: "EU Privacy Law" }),
    ).toBeVisible();
    expect(
      screen.getByRole("heading", { name: "Brazil Privacy Law" }),
    ).toBeVisible();
    const [requestedUrl, requestInit] = fetchResponse.mock.calls[0] ?? [];
    expect(requestedUrl).toBe("/api/v1/corpora?includeDisabled=true");
    expect(requestInit?.signal).toBeInstanceOf(AbortSignal);
    await expectNoAccessibilityViolations(result.container);
  });

  it("shows an actionable failure without rendering demonstration data", async () => {
    const fetchResponse = vi
      .fn<typeof fetch>()
      .mockRejectedValue(new Error("offline"));

    renderAtRoute(
      <CorpusCatalogPage
        provider={createHttpResearchProvider({ fetch: fetchResponse })}
      />,
    );

    expect(await screen.findByRole("alert")).toBeVisible();
    expect(
      screen.queryByText("European Data Protection Framework"),
    ).not.toBeInTheDocument();
    expect(screen.queryAllByRole("article")).toHaveLength(0);
  });

  it("keeps creation and editing in dedicated navigation destinations", async () => {
    const existing = corpus(
      "10000000-0000-4000-8000-000000000004",
      "Commercial law",
      "en",
    );
    const provider = new ManagementProvider([existing]);
    renderAtRoute(<CorpusCatalogPage provider={provider} />);
    await screen.findByRole("heading", { name: "Commercial law" });

    expect(screen.getByRole("link", { name: "Create corpus" })).toHaveAttribute(
      "href",
      "/corpora/new",
    );
    expect(
      screen.getByRole("link", { name: "Edit corpus Commercial law" }),
    ).toHaveAttribute(
      "href",
      "/corpora/10000000-0000-4000-8000-000000000004/edit",
    );
    expect(
      screen.queryByRole("textbox", { name: "Name" }),
    ).not.toBeInTheDocument();
  });

  it("disables a corpus while preserving its identity", async () => {
    const user = userEvent.setup();
    const existing = corpus(
      "10000000-0000-4000-8000-000000000004",
      "Commercial law",
      "en",
    );
    const provider = new ManagementProvider([existing]);
    const confirmation = vi.spyOn(globalThis, "confirm").mockReturnValue(true);
    renderAtRoute(<CorpusCatalogPage provider={provider} />);
    await screen.findByRole("heading", { name: "Commercial law" });

    await user.click(screen.getByLabelText("Manage corpus Commercial law"));
    await user.click(screen.getByRole("button", { name: "Disable corpus" }));
    expect(await screen.findByText("Disabled")).toBeVisible();
    expect(provider.disabledVersion).toBe(1);
    expect(confirmation).toHaveBeenCalledWith(
      "Disable Commercial law? Its documents and ingestion history will be preserved.",
    );
    confirmation.mockRestore();
  });
});

class ManagementProvider implements ResearchProvider {
  createdDraft: CorpusDraft | undefined;
  updatedVersion: number | undefined;
  disabledVersion: number | undefined;

  constructor(
    private corpora: CorpusResponse[] = [],
    private readonly failUpdate = false,
  ) {}

  listCorpora(): Promise<readonly CorpusResponse[]> {
    return Promise.resolve(this.corpora);
  }

  createCorpus(draft: CorpusDraft): Promise<CorpusResponse> {
    this.createdDraft = draft;
    const created = corpus(
      "10000000-0000-4000-8000-000000000003",
      draft.name,
      draft.language,
    );
    this.corpora = [...this.corpora, created];
    return Promise.resolve(created);
  }

  createUrlSource(): Promise<never> {
    throw new Error("not used");
  }

  createPdfSource(): Promise<never> {
    throw new Error("not used");
  }

  retrySource(): Promise<never> {
    throw new Error("not used");
  }

  reprocessSource(): Promise<never> {
    throw new Error("not used");
  }

  publishSnapshot(): Promise<never> {
    throw new Error("not used");
  }

  getCorpus(): Promise<CorpusResponse> {
    throw new Error("not used");
  }

  getCorpusOpeningSuggestions(): Promise<never> {
    throw new Error("not used");
  }

  listSources(): Promise<never[]> {
    return Promise.resolve([]);
  }

  listSnapshots(): Promise<never[]> {
    return Promise.resolve([]);
  }

  getGraphRelease(): Promise<never> {
    throw new Error("not used");
  }

  getDocument(): Promise<never> {
    throw new Error("not used");
  }

  getDocumentVersion(): Promise<never> {
    throw new Error("not used");
  }

  updateCorpus(
    corpusId: string,
    update: CorpusUpdate,
  ): Promise<CorpusResponse> {
    if (this.failUpdate) {
      return Promise.reject(
        new ResearchRequestError(
          "stale_state",
          "The corpus changed; reload and retry.",
          undefined,
          "10000000-0000-4000-8000-000000000099",
        ),
      );
    }
    this.updatedVersion = update.version;
    const current = this.requiredCorpus(corpusId);
    const updated = {
      ...current,
      ...update,
      version: current.version + 1,
    };
    this.replace(updated);
    return Promise.resolve(updated);
  }

  disableCorpus(corpusId: string, version: number): Promise<CorpusResponse> {
    this.disabledVersion = version;
    return Promise.resolve(this.setStatus(corpusId, "disabled"));
  }

  enableCorpus(corpusId: string): Promise<CorpusResponse> {
    return Promise.resolve(this.setStatus(corpusId, "enabled"));
  }

  private setStatus(
    corpusId: string,
    status: CorpusResponse["status"],
  ): CorpusResponse {
    const current = this.requiredCorpus(corpusId);
    const updated = { ...current, status, version: current.version + 1 };
    this.replace(updated);
    return updated;
  }

  private requiredCorpus(corpusId: string): CorpusResponse {
    const found = this.corpora.find((item) => item.id === corpusId);
    if (!found) throw new Error("test corpus not found");
    return found;
  }

  private replace(corpusValue: CorpusResponse): void {
    this.corpora = this.corpora.map((item) =>
      item.id === corpusValue.id ? corpusValue : item,
    );
  }
}

function corpus(
  id: string,
  name: string,
  language: "en" | "pt",
): CorpusResponse {
  return {
    id,
    name,
    description: `${name} official materials.`,
    language,
    jurisdiction: language === "en" ? "European Union" : "Brazil",
    status: "enabled",
    sourceCount: 1,
    version: 1,
    createdAt: "2026-08-17T12:00:00Z",
    updatedAt: "2026-08-17T12:00:00Z",
  };
}

function jsonResponse(value: unknown): Response {
  return new Response(JSON.stringify(value), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}
