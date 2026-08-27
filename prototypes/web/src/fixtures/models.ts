export type LanguageCode = "en" | "pt";
export type SourceKind = "pdf" | "external";
export type SourceStatus = "available" | "unavailable";
export type WorkspaceMode = "chat" | "source";

export interface SourceSection {
  readonly id: string;
  readonly marker: string;
  readonly heading: string;
  readonly paragraphs: readonly string[];
}

export interface CorpusSource {
  readonly id: string;
  readonly corpusId: string;
  readonly kind: SourceKind;
  readonly title: string;
  readonly shortTitle: string;
  readonly publisher: string;
  readonly publishedLabel: string;
  readonly status: SourceStatus;
  readonly externalUrl?: string;
  readonly sections: readonly SourceSection[];
}

export interface CitationFixture {
  readonly id: string;
  readonly sourceId: string;
  readonly sectionId: string;
  readonly label: string;
}

export interface PreparedAnswer {
  readonly id: string;
  readonly prompts: readonly string[];
  readonly answer: string;
  readonly citations: readonly CitationFixture[];
}

export interface CorpusReleaseFixture {
  readonly snapshotId: string;
  readonly snapshotManifestSha256: string;
}

export interface OpeningSuggestionFixture {
  readonly caseId: string;
  readonly rank: number;
  readonly language: LanguageCode;
  readonly question: string;
}

export interface OpeningSuggestionSetFixture {
  readonly corpusId: string;
  readonly snapshotId: string;
  readonly snapshotManifestSha256: string;
  readonly suggestions: readonly OpeningSuggestionFixture[];
}

export interface CorpusFixture {
  readonly id: string;
  readonly label: string;
  readonly eyebrowKey: string;
  readonly descriptionKey: string;
  readonly language: LanguageCode;
  readonly jurisdiction: string;
  readonly sources: readonly CorpusSource[];
  readonly preparedAnswers: readonly PreparedAnswer[];
  readonly activeRelease?: CorpusReleaseFixture;
  readonly openingSuggestionSet?: OpeningSuggestionSetFixture;
  readonly failurePrompts: readonly string[];
}

export interface ViewerState {
  readonly sourceId: string | null;
  readonly sectionId: string | null;
}
