import {
  assertEvaluationDatasetPreflightRequest,
  assertEvaluationDatasetRevisionId,
  parseEvaluationCatalogErrorEnvelope,
  parseEvaluationDatasetCatalog,
  parseEvaluationDatasetDetail,
  parseEvaluationDatasetPreflightResponse,
  type EvaluationCatalogErrorCode,
  type EvaluationDatasetCatalogEntry,
  type EvaluationDatasetDetail,
  type EvaluationDatasetPreflightRequest,
  type EvaluationDatasetPreflightResponse,
  type EvaluationMissingRequirement,
} from "./evaluationCatalog";

interface HttpEvaluationCatalogClientOptions {
  readonly baseUrl?: string;
  readonly fetch?: typeof fetch;
}

export class EvaluationCatalogRequestError extends Error {
  constructor(
    readonly code: EvaluationCatalogErrorCode,
    message: string,
    readonly requestId: string,
    readonly missingRequirements:
      readonly EvaluationMissingRequirement[] | undefined,
  ) {
    super(message);
    this.name = "EvaluationCatalogRequestError";
  }
}

export interface EvaluationCatalogClient {
  listDatasets(
    signal: AbortSignal,
  ): Promise<readonly EvaluationDatasetCatalogEntry[]>;
  getDataset(
    datasetRevisionId: string,
    signal: AbortSignal,
  ): Promise<EvaluationDatasetDetail>;
  preflightDataset(
    request: EvaluationDatasetPreflightRequest,
    signal: AbortSignal,
  ): Promise<EvaluationDatasetPreflightResponse>;
}

class HttpEvaluationCatalogClient implements EvaluationCatalogClient {
  readonly #baseUrl: string;
  readonly #fetch: typeof fetch;

  constructor(options: HttpEvaluationCatalogClientOptions) {
    this.#baseUrl = options.baseUrl ?? "/api/v1";
    this.#fetch = options.fetch ?? globalThis.fetch.bind(globalThis);
  }

  async listDatasets(
    signal: AbortSignal,
  ): Promise<readonly EvaluationDatasetCatalogEntry[]> {
    return parseEvaluationDatasetCatalog(
      await this.#request(`${this.#baseUrl}/evaluation-datasets`, signal),
    );
  }

  async getDataset(
    datasetRevisionId: string,
    signal: AbortSignal,
  ): Promise<EvaluationDatasetDetail> {
    assertEvaluationDatasetRevisionId(datasetRevisionId);
    const detail = parseEvaluationDatasetDetail(
      await this.#request(
        `${this.#baseUrl}/evaluation-datasets/${encodeURIComponent(datasetRevisionId)}`,
        signal,
      ),
    );
    if (detail.revision.id !== datasetRevisionId) {
      throw new Error(
        "Evaluation dataset detail identity does not match request.",
      );
    }
    return detail;
  }

  async preflightDataset(
    request: EvaluationDatasetPreflightRequest,
    signal: AbortSignal,
  ): Promise<EvaluationDatasetPreflightResponse> {
    assertEvaluationDatasetPreflightRequest(request);
    const query = new URLSearchParams({
      corpusId: request.corpusId,
      snapshotId: request.snapshotId,
    });
    const response = parseEvaluationDatasetPreflightResponse(
      await this.#request(
        `${this.#baseUrl}/evaluation-datasets/${encodeURIComponent(request.datasetRevisionId)}/preflight?${query.toString()}`,
        signal,
      ),
    );
    if (
      response.datasetRevisionId !== request.datasetRevisionId ||
      response.corpusId !== request.corpusId ||
      response.snapshotId !== request.snapshotId
    ) {
      throw new Error(
        "Evaluation dataset preflight identities do not match the requested immutable selection.",
      );
    }
    return response;
  }

  async #request(url: string, signal: AbortSignal): Promise<unknown> {
    const response = await this.#fetch(url, {
      method: "GET",
      signal,
      credentials: "include",
    });
    if (!response.ok) {
      await this.#throwResponseError(response);
    }
    return response.json() as Promise<unknown>;
  }

  async #throwResponseError(response: Response): Promise<never> {
    const envelope = parseEvaluationCatalogErrorEnvelope(await response.json());
    throw new EvaluationCatalogRequestError(
      envelope.error.code,
      envelope.error.message,
      envelope.error.requestId,
      envelope.error.missingRequirements,
    );
  }
}

export function createEvaluationCatalogClient(
  options: HttpEvaluationCatalogClientOptions = {},
): EvaluationCatalogClient {
  return new HttpEvaluationCatalogClient(options);
}
