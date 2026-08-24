import {
  parseCorpusList,
  parseCorpusResponse,
  parseDocumentResponse,
  parseGraphReleaseResponse,
  parseErrorEnvelope,
  parseSnapshotList,
  parseSnapshotPublicationResponse,
  parseSourceList,
  type CorpusResponse,
  type DocumentResponse,
  type GraphReleaseResponse,
  type PublicErrorCode,
  type SnapshotPublicationResponse,
  type SnapshotResponse,
  type SourceResponse,
} from "./contract";
import type {
  CorpusDraft,
  CorpusUpdate,
  ResearchProvider,
  UrlSourceDraft,
} from "../research/domain/authoritative";

interface HttpResearchProviderOptions {
  readonly baseUrl?: string;
  readonly fetch?: typeof fetch;
}

export class ResearchRequestError extends Error {
  constructor(
    readonly code: PublicErrorCode,
    message: string,
    readonly fields: Readonly<Record<string, string>> | undefined,
    readonly requestId: string,
  ) {
    super(message);
    this.name = "ResearchRequestError";
  }
}

class HttpResearchProvider implements ResearchProvider {
  readonly #baseUrl: string;
  readonly #fetch: typeof fetch;

  constructor(options: HttpResearchProviderOptions) {
    this.#baseUrl = options.baseUrl ?? "/api/v1";
    this.#fetch = options.fetch ?? globalThis.fetch.bind(globalThis);
  }

  async listCorpora(
    signal: AbortSignal,
    includeDisabled = false,
  ): Promise<readonly CorpusResponse[]> {
    const query = includeDisabled ? "?includeDisabled=true" : "";
    return parseCorpusList(
      await this.#request(`${this.#baseUrl}/corpora${query}`, signal),
    );
  }

  async getCorpus(
    corpusId: string,
    signal: AbortSignal,
  ): Promise<CorpusResponse> {
    return parseCorpusResponse(
      await this.#request(
        `${this.#baseUrl}/corpora/${encodeURIComponent(corpusId)}`,
        signal,
      ),
    );
  }

  async listSources(
    corpusId: string,
    signal: AbortSignal,
  ): Promise<readonly SourceResponse[]> {
    return parseSourceList(
      await this.#request(
        `${this.#baseUrl}/corpora/${encodeURIComponent(corpusId)}/sources`,
        signal,
      ),
    );
  }

  async listSnapshots(
    corpusId: string,
    signal: AbortSignal,
  ): Promise<readonly SnapshotResponse[]> {
    return parseSnapshotList(
      await this.#request(
        `${this.#baseUrl}/corpora/${encodeURIComponent(corpusId)}/snapshots`,
        signal,
      ),
    );
  }

  async getGraphRelease(
    corpusId: string,
    snapshotId: string,
    signal: AbortSignal,
  ): Promise<GraphReleaseResponse> {
    return parseGraphReleaseResponse(
      await this.#request(
        `${this.#baseUrl}/corpora/${encodeURIComponent(corpusId)}/snapshots/${encodeURIComponent(snapshotId)}/graph-release`,
        signal,
      ),
    );
  }

  async getDocument(
    corpusId: string,
    sourceId: string,
    signal: AbortSignal,
  ): Promise<DocumentResponse> {
    return parseDocumentResponse(
      await this.#request(
        `${this.#baseUrl}/corpora/${encodeURIComponent(corpusId)}/sources/${encodeURIComponent(sourceId)}/document`,
        signal,
      ),
    );
  }

  async getDocumentVersion(
    corpusId: string,
    sourceId: string,
    documentVersionId: string,
    signal: AbortSignal,
  ): Promise<DocumentResponse> {
    return parseDocumentResponse(
      await this.#request(
        `${this.#baseUrl}/corpora/${encodeURIComponent(corpusId)}/sources/${encodeURIComponent(sourceId)}/documents/${encodeURIComponent(documentVersionId)}`,
        signal,
      ),
    );
  }

  async createCorpus(
    draft: CorpusDraft,
    signal: AbortSignal,
  ): Promise<CorpusResponse> {
    return parseCorpusResponse(
      await this.#request(`${this.#baseUrl}/corpora`, signal, "POST", draft),
    );
  }

  async createUrlSource(
    corpusId: string,
    draft: UrlSourceDraft,
    signal: AbortSignal,
  ): Promise<SourceResponse> {
    return parseSourceList([
      await this.#request(
        `${this.#baseUrl}/corpora/${encodeURIComponent(corpusId)}/sources/url`,
        signal,
        "POST",
        draft,
      ),
    ])[0] as SourceResponse;
  }

  async createPdfSource(
    corpusId: string,
    title: string,
    file: File,
    signal: AbortSignal,
  ): Promise<SourceResponse> {
    const body = new FormData();
    body.append("title", title);
    body.append("file", file);
    const response = await this.#fetch(
      `${this.#baseUrl}/corpora/${encodeURIComponent(corpusId)}/sources/pdf`,
      { method: "POST", body, signal },
    );
    if (!response.ok) await this.#throwResponseError(response);
    return parseSourceList([await response.json()])[0] as SourceResponse;
  }

  async retrySource(
    corpusId: string,
    sourceId: string,
    version: number,
    signal: AbortSignal,
  ): Promise<SourceResponse> {
    return this.#sourceLifecycle(corpusId, sourceId, "retry", version, signal);
  }

  async reprocessSource(
    corpusId: string,
    sourceId: string,
    version: number,
    signal: AbortSignal,
  ): Promise<SourceResponse> {
    return this.#sourceLifecycle(
      corpusId,
      sourceId,
      "reprocess",
      version,
      signal,
    );
  }

  async publishSnapshot(
    corpusId: string,
    sourceId: string,
    documentId: string,
    expectedReleaseVersion: number,
    signal: AbortSignal,
  ): Promise<SnapshotPublicationResponse> {
    return parseSnapshotPublicationResponse(
      await this.#request(
        `${this.#baseUrl}/corpora/${encodeURIComponent(corpusId)}/snapshots`,
        signal,
        "POST",
        { sourceId, documentId, expectedReleaseVersion },
      ),
    );
  }

  async #sourceLifecycle(
    corpusId: string,
    sourceId: string,
    action: "retry" | "reprocess",
    version: number,
    signal: AbortSignal,
  ): Promise<SourceResponse> {
    return parseSourceList([
      await this.#request(
        `${this.#baseUrl}/corpora/${encodeURIComponent(corpusId)}/sources/${encodeURIComponent(sourceId)}/${action}`,
        signal,
        "POST",
        { version },
      ),
    ])[0] as SourceResponse;
  }

  async updateCorpus(
    corpusId: string,
    update: CorpusUpdate,
    signal: AbortSignal,
  ): Promise<CorpusResponse> {
    return parseCorpusResponse(
      await this.#request(
        `${this.#baseUrl}/corpora/${encodeURIComponent(corpusId)}`,
        signal,
        "PATCH",
        update,
      ),
    );
  }

  async disableCorpus(
    corpusId: string,
    version: number,
    signal: AbortSignal,
  ): Promise<CorpusResponse> {
    return this.#lifecycle(corpusId, "disable", version, signal);
  }

  async enableCorpus(
    corpusId: string,
    version: number,
    signal: AbortSignal,
  ): Promise<CorpusResponse> {
    return this.#lifecycle(corpusId, "enable", version, signal);
  }

  async #lifecycle(
    corpusId: string,
    action: "disable" | "enable",
    version: number,
    signal: AbortSignal,
  ): Promise<CorpusResponse> {
    return parseCorpusResponse(
      await this.#request(
        `${this.#baseUrl}/corpora/${encodeURIComponent(corpusId)}/${action}`,
        signal,
        "POST",
        { version },
      ),
    );
  }

  async #request(
    url: string,
    signal: AbortSignal,
    method = "GET",
    body?: object,
  ): Promise<unknown> {
    const response = await this.#fetch(url, {
      signal,
      method,
      ...(body === undefined
        ? {}
        : {
            body: JSON.stringify(body),
            headers: { "Content-Type": "application/json" },
          }),
    });
    if (!response.ok) {
      await this.#throwResponseError(response);
    }
    return response.json() as Promise<unknown>;
  }

  async #throwResponseError(response: Response): Promise<never> {
    const envelope = parseErrorEnvelope(await response.json());
    throw new ResearchRequestError(
      envelope.error.code,
      envelope.error.message,
      envelope.error.fields,
      envelope.error.requestId,
    );
  }
}

export function createHttpResearchProvider(
  options: HttpResearchProviderOptions = {},
): ResearchProvider {
  return new HttpResearchProvider(options);
}
