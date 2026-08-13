import { describe, expect, it } from "vitest";

import type { Corpus, Source } from "./models";
import { createResearchCatalog } from "./researchCatalog";

const emptyCorpus = (id: string): Corpus => ({
  id,
  language: "en",
  name: "Reference law",
  jurisdiction: "Example jurisdiction",
  summary: "A deterministic legal research collection.",
  sources: [],
  suggestedQuestions: [],
  preparedResponses: [],
});

const pdfSource = (corpusId: string, id: string): Source => ({
  id,
  corpusId,
  kind: "pdf",
  title: "Reference document",
  authority: "Reference authority",
  officialReference: "REF-1",
  pageCount: 1,
  locations: [{ id: "page-1", label: "Page 1", content: "Evidence." }],
});

describe("research catalog", () => {
  it("rejects duplicate corpus identifiers", () => {
    expect(() =>
      createResearchCatalog([emptyCorpus("privacy"), emptyCorpus("privacy")]),
    ).toThrow("Corpus identifiers must be unique.");
  });

  it("rejects external source URLs that do not use HTTPS", () => {
    const source: Source = {
      id: "official-link",
      corpusId: "privacy",
      kind: "external",
      title: "Official source",
      authority: "Reference authority",
      officialReference: "REF-2",
      locations: [],
      url: "http://example.com/legal-source",
      preview: { status: "unavailable", reason: "Preview unavailable." },
    };

    expect(() =>
      createResearchCatalog([{ ...emptyCorpus("privacy"), sources: [source] }]),
    ).toThrow("External source URLs must use HTTPS.");
  });

  it("rejects prepared citations outside their corpus", () => {
    const first = {
      ...emptyCorpus("first"),
      sources: [pdfSource("first", "first-source")],
      preparedResponses: [
        {
          id: "cross-corpus-response",
          prompts: ["question"],
          outcome: "answered" as const,
          parts: [
            {
              type: "citation" as const,
              citation: {
                id: "citation",
                sourceId: "second-source",
                locationId: "page-1",
                label: "Foreign evidence",
              },
            },
          ],
        },
      ],
    };
    const second = {
      ...emptyCorpus("second"),
      sources: [pdfSource("second", "second-source")],
    };

    expect(() => createResearchCatalog([first, second])).toThrow(
      "Prepared response citations must resolve inside their corpus.",
    );
  });
});
