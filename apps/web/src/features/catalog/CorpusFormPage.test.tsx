import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import type {
  CorpusResponse,
  DocumentResponse,
  SourceResponse,
} from "../../api/contract";
import { ResearchRequestError } from "../../api/researchProvider";
import { AppRoutes } from "../../app/routes";
import type {
  CorpusDraft,
  CorpusUpdate,
  ResearchProvider,
} from "../../research/domain/authoritative";
import {
  expectNoAccessibilityViolations,
  renderAtRoute,
} from "../../test/render";

describe("corpus form page", () => {
  it("creates a corpus on its own accessible route", async () => {
    const user = userEvent.setup();
    const provider = new FormProvider();
    const result = renderAtRoute(
      <AppRoutes provider={provider} />,
      "/corpora/new",
    );

    expect(
      await screen.findByRole("heading", { name: "Create a legal corpus" }),
    ).toBeVisible();
    await user.type(
      screen.getByRole("textbox", { name: "Name" }),
      "Employment law",
    );
    await user.type(
      screen.getByRole("textbox", { name: "Description" }),
      "Official employment materials.",
    );
    await user.type(
      screen.getByRole("textbox", { name: "Jurisdiction" }),
      "United States",
    );
    await user.click(screen.getByRole("button", { name: "Create corpus" }));

    expect(
      await screen.findByRole("heading", { name: "Employment law" }),
    ).toBeVisible();
    expect(provider.createdDraft).toEqual({
      name: "Employment law",
      description: "Official employment materials.",
      language: "en",
      jurisdiction: "United States",
    });
    await expectNoAccessibilityViolations(result.container);
  });

  it("loads and updates an existing corpus without disturbing the catalog", async () => {
    const user = userEvent.setup();
    const existing = corpus("Commercial law");
    const provider = new FormProvider([existing]);
    renderAtRoute(
      <AppRoutes provider={provider} />,
      `/corpora/${existing.id}/edit`,
    );

    const name = await screen.findByRole("textbox", { name: "Name" });
    expect(name).toHaveValue("Commercial law");
    await user.clear(name);
    await user.type(name, "Updated commercial law");
    await user.click(screen.getByRole("button", { name: "Save changes" }));

    expect(
      await screen.findByRole("heading", { name: "Updated commercial law" }),
    ).toBeVisible();
    expect(provider.updatedVersion).toBe(1);
  });

  it("preserves entered values when an edit conflicts with a newer version", async () => {
    const user = userEvent.setup();
    const existing = corpus("Tax law");
    const provider = new FormProvider([existing], true);
    renderAtRoute(
      <AppRoutes provider={provider} />,
      `/corpora/${existing.id}/edit`,
    );

    const name = await screen.findByRole("textbox", { name: "Name" });
    await user.clear(name);
    await user.type(name, "Updated tax law");
    await user.click(screen.getByRole("button", { name: "Save changes" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "This corpus changed after you opened it. Reload before saving again.",
    );
    expect(screen.getByRole("textbox", { name: "Name" })).toHaveValue(
      "Updated tax law",
    );
  });
});

class FormProvider implements ResearchProvider {
  createdDraft: CorpusDraft | undefined;
  updatedVersion: number | undefined;

  constructor(
    private corpora: CorpusResponse[] = [],
    private readonly failUpdate = false,
  ) {}

  listCorpora(): Promise<readonly CorpusResponse[]> {
    return Promise.resolve(this.corpora);
  }

  getCorpus(corpusId: string): Promise<CorpusResponse> {
    const found = this.corpora.find((item) => item.id === corpusId);
    return found
      ? Promise.resolve(found)
      : Promise.reject(new Error("not found"));
  }

  createCorpus(draft: CorpusDraft): Promise<CorpusResponse> {
    this.createdDraft = draft;
    const created = { ...corpus(draft.name), ...draft };
    this.corpora = [...this.corpora, created];
    return Promise.resolve(created);
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
    const current = this.corpora.find((item) => item.id === corpusId);
    if (!current) return Promise.reject(new Error("not found"));
    const updated = { ...current, ...update, version: current.version + 1 };
    this.corpora = this.corpora.map((item) =>
      item.id === corpusId ? updated : item,
    );
    return Promise.resolve(updated);
  }

  createUrlSource(): Promise<SourceResponse> {
    return Promise.reject(new Error("not used"));
  }
  createPdfSource(): Promise<SourceResponse> {
    return Promise.reject(new Error("not used"));
  }
  retrySource(): Promise<SourceResponse> {
    return Promise.reject(new Error("not used"));
  }
  reprocessSource(): Promise<SourceResponse> {
    return Promise.reject(new Error("not used"));
  }
  publishSnapshot(): Promise<never> {
    return Promise.reject(new Error("not used"));
  }
  listSources(): Promise<readonly SourceResponse[]> {
    return Promise.resolve([]);
  }
  listSnapshots(): Promise<never[]> {
    return Promise.resolve([]);
  }
  getGraphRelease(): Promise<never> {
    return Promise.reject(new Error("not used"));
  }
  getDocument(): Promise<DocumentResponse> {
    return Promise.reject(new Error("not used"));
  }
  getDocumentVersion(): Promise<DocumentResponse> {
    return Promise.reject(new Error("not used"));
  }
  disableCorpus(): Promise<CorpusResponse> {
    return Promise.reject(new Error("not used"));
  }
  enableCorpus(): Promise<CorpusResponse> {
    return Promise.reject(new Error("not used"));
  }
}

function corpus(name: string): CorpusResponse {
  return {
    id: "10000000-0000-4000-8000-000000000004",
    name,
    description: "Official commercial law materials.",
    language: "en",
    jurisdiction: "European Union",
    status: "enabled",
    sourceCount: 1,
    version: 1,
    createdAt: "2026-08-17T12:00:00Z",
    updatedAt: "2026-08-17T12:00:00Z",
  };
}
