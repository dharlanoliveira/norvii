import type {
  CorpusLanguage,
  CorpusOpeningSuggestionResponse,
  CorpusResponse,
  DocumentResponse,
  GraphReleaseResponse,
  SnapshotPublicationResponse,
  SnapshotResponse,
  SourceResponse,
} from "../../api/contract";

export interface ResearchProvider {
  listCorpora(
    signal: AbortSignal,
    includeDisabled?: boolean,
  ): Promise<readonly CorpusResponse[]>;
  getCorpus(corpusId: string, signal: AbortSignal): Promise<CorpusResponse>;
  getCorpusOpeningSuggestions(
    corpusId: string,
    interfaceLanguage: CorpusLanguage,
    signal: AbortSignal,
  ): Promise<CorpusOpeningSuggestionResponse>;
  listSources(
    corpusId: string,
    signal: AbortSignal,
  ): Promise<readonly SourceResponse[]>;
  listSnapshots(
    corpusId: string,
    signal: AbortSignal,
  ): Promise<readonly SnapshotResponse[]>;
  getGraphRelease(
    corpusId: string,
    snapshotId: string,
    signal: AbortSignal,
  ): Promise<GraphReleaseResponse>;
  getDocument(
    corpusId: string,
    sourceId: string,
    signal: AbortSignal,
  ): Promise<DocumentResponse>;
  getDocumentVersion(
    corpusId: string,
    sourceId: string,
    documentVersionId: string,
    signal: AbortSignal,
  ): Promise<DocumentResponse>;
  createUrlSource(
    corpusId: string,
    draft: UrlSourceDraft,
    signal: AbortSignal,
  ): Promise<SourceResponse>;
  createPdfSource(
    corpusId: string,
    title: string,
    file: File,
    signal: AbortSignal,
  ): Promise<SourceResponse>;
  retrySource(
    corpusId: string,
    sourceId: string,
    version: number,
    signal: AbortSignal,
  ): Promise<SourceResponse>;
  reprocessSource(
    corpusId: string,
    sourceId: string,
    version: number,
    signal: AbortSignal,
  ): Promise<SourceResponse>;
  publishSnapshot(
    corpusId: string,
    sourceId: string,
    documentId: string,
    expectedReleaseVersion: number,
    signal: AbortSignal,
  ): Promise<SnapshotPublicationResponse>;
  createCorpus(
    draft: CorpusDraft,
    signal: AbortSignal,
  ): Promise<CorpusResponse>;
  updateCorpus(
    corpusId: string,
    update: CorpusUpdate,
    signal: AbortSignal,
  ): Promise<CorpusResponse>;
  disableCorpus(
    corpusId: string,
    version: number,
    signal: AbortSignal,
  ): Promise<CorpusResponse>;
  enableCorpus(
    corpusId: string,
    version: number,
    signal: AbortSignal,
  ): Promise<CorpusResponse>;
}

export interface UrlSourceDraft {
  readonly title: string;
  readonly url: string;
}

export interface CorpusDraft {
  readonly name: string;
  readonly description: string;
  readonly language: "en" | "pt";
  readonly jurisdiction: string;
}

export interface CorpusUpdate extends CorpusDraft {
  readonly version: number;
}
