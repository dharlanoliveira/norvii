import {
  assertEvaluationRunCaseId,
  assertEvaluationRunId,
  parseEvaluationComparison,
  parseEvaluationRunCase,
  parseEvaluationRunSummary,
  type EvaluationComparison,
  type EvaluationRunCase,
  type EvaluationRunSummary,
} from "./evaluationRun";

interface HttpEvaluationRunClientOptions {
  readonly baseUrl?: string;
  readonly fetch?: typeof fetch;
}

export interface EvaluationRunClient {
  getRun(runId: string, signal: AbortSignal): Promise<EvaluationRunSummary>;
  getCase(
    runId: string,
    caseId: string,
    signal: AbortSignal,
  ): Promise<EvaluationRunCase>;
  compareRuns(
    leftRunId: string,
    rightRunId: string,
    signal: AbortSignal,
  ): Promise<EvaluationComparison>;
}

class HttpEvaluationRunClient implements EvaluationRunClient {
  readonly #baseUrl: string;
  readonly #fetch: typeof fetch;

  constructor(options: HttpEvaluationRunClientOptions) {
    this.#baseUrl = options.baseUrl ?? "/api/v1";
    this.#fetch = options.fetch ?? globalThis.fetch.bind(globalThis);
  }

  async getRun(
    runId: string,
    signal: AbortSignal,
  ): Promise<EvaluationRunSummary> {
    assertEvaluationRunId(runId);
    const result = parseEvaluationRunSummary(
      await this.#request(
        `${this.#baseUrl}/evaluations/${encodeURIComponent(runId)}`,
        signal,
      ),
    );
    if (result.id !== runId)
      throw new Error("Evaluation run identity does not match request.");
    return result;
  }

  async getCase(
    runId: string,
    caseId: string,
    signal: AbortSignal,
  ): Promise<EvaluationRunCase> {
    assertEvaluationRunId(runId);
    assertEvaluationRunCaseId(caseId);
    const result = parseEvaluationRunCase(
      await this.#request(
        `${this.#baseUrl}/evaluations/${encodeURIComponent(runId)}/cases/${encodeURIComponent(caseId)}`,
        signal,
      ),
    );
    if (result.runId !== runId || result.id !== caseId)
      throw new Error("Evaluation case identity does not match request.");
    return result;
  }

  async compareRuns(
    leftRunId: string,
    rightRunId: string,
    signal: AbortSignal,
  ): Promise<EvaluationComparison> {
    assertEvaluationRunId(leftRunId);
    assertEvaluationRunId(rightRunId);
    const query = new URLSearchParams({ left: leftRunId, right: rightRunId });
    const result = parseEvaluationComparison(
      await this.#request(
        `${this.#baseUrl}/evaluations/compare?${query.toString()}`,
        signal,
      ),
    );
    if (result.leftRunId !== leftRunId || result.rightRunId !== rightRunId)
      throw new Error("Evaluation comparison identities do not match request.");
    return result;
  }

  async #request(url: string, signal: AbortSignal): Promise<unknown> {
    const response = await this.#fetch(url, {
      method: "GET",
      signal,
      // The same-origin maintainer BFF authenticates this request with its
      // HttpOnly session cookie. Browser code must not receive bearer tokens.
      credentials: "include",
    });
    if (!response.ok) throw new Error("Evaluation inspection request failed.");
    return response.json() as Promise<unknown>;
  }
}

export function createEvaluationRunClient(
  options: HttpEvaluationRunClientOptions = {},
): EvaluationRunClient {
  return new HttpEvaluationRunClient(options);
}
