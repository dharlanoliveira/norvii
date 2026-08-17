import type {
  Citation,
  Corpus,
  CorpusSummary,
  ResolvedCitation,
  Source,
} from "./models";

export interface ResearchCatalog {
  listCorpora(): readonly CorpusSummary[];
  findCorpus(corpusId: string): Corpus | undefined;
  resolveCitation(
    corpusId: string,
    citation: Citation,
  ): ResolvedCitation | undefined;
}

function validateUniqueIdentifiers(corpora: readonly Corpus[]): void {
  const corpusIdentifiers = new Set<string>();

  for (const corpus of corpora) {
    if (corpusIdentifiers.has(corpus.id)) {
      throw new Error("Corpus identifiers must be unique.");
    }
    corpusIdentifiers.add(corpus.id);

    const sourceIdentifiers = new Set<string>();
    for (const source of corpus.sources) {
      if (sourceIdentifiers.has(source.id)) {
        throw new Error("Source identifiers must be unique within a corpus.");
      }
      sourceIdentifiers.add(source.id);
    }
  }
}

function validateSourceOwnership(corpora: readonly Corpus[]): void {
  for (const corpus of corpora) {
    for (const source of corpus.sources) {
      if (source.corpusId !== corpus.id) {
        throw new Error("Every source must belong to its containing corpus.");
      }
    }
  }
}

function validateExternalLinks(corpora: readonly Corpus[]): void {
  const externalSources = corpora.flatMap((corpus) =>
    corpus.sources.filter(
      (source): source is Extract<Source, { kind: "external" }> =>
        source.kind === "external",
    ),
  );

  for (const source of externalSources) {
    if (new URL(source.url).protocol !== "https:") {
      throw new Error("External source URLs must use HTTPS.");
    }
  }
}

function validatePreparedCitations(corpora: readonly Corpus[]): void {
  for (const corpus of corpora) {
    for (const response of corpus.preparedResponses) {
      for (const part of response.parts) {
        if (part.type !== "citation") continue;

        const source = corpus.sources.find(
          (candidate) => candidate.id === part.citation.sourceId,
        );
        const location = source?.locations.find(
          (candidate) => candidate.id === part.citation.locationId,
        );

        if (!source || !location) {
          throw new Error(
            "Prepared response citations must resolve inside their corpus.",
          );
        }
      }
    }
  }
}

export function createResearchCatalog(
  corpora: readonly Corpus[],
): ResearchCatalog {
  validateUniqueIdentifiers(corpora);
  validateSourceOwnership(corpora);
  validateExternalLinks(corpora);
  validatePreparedCitations(corpora);

  const corpusById = new Map(corpora.map((corpus) => [corpus.id, corpus]));

  return {
    listCorpora: () =>
      corpora.map(({ id, language, name, jurisdiction, summary, sources }) => ({
        id,
        language,
        name,
        jurisdiction,
        summary,
        sourceCount: sources.length,
      })),
    findCorpus: (corpusId) => corpusById.get(corpusId),
    resolveCitation: (corpusId, citation) => {
      const corpus = corpusById.get(corpusId);
      const source = corpus?.sources.find(
        (candidate) => candidate.id === citation.sourceId,
      );
      const location = source?.locations.find(
        (candidate) => candidate.id === citation.locationId,
      );

      return source && location ? { source, location } : undefined;
    },
  };
}
