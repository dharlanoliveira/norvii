import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import comparisonFixture from "../../../../../contracts/evaluation/v1/fixtures/evaluation-comparison-response.json?raw";
import nonComparableFixture from "../../../../../contracts/evaluation/v1/fixtures/evaluation-comparison-non-comparable-response.json?raw";
import caseFixture from "../../../../../contracts/evaluation/v1/fixtures/evaluation-run-case-response.json?raw";
import summaryFixture from "../../../../../contracts/evaluation/v1/fixtures/evaluation-run-summary-response.json?raw";
import {
  parseEvaluationComparison,
  parseEvaluationRunCase,
  parseEvaluationRunSummary,
  type EvaluationComparison,
  type EvaluationRunCase,
  type EvaluationRunSummary,
} from "../../api/evaluationRun";
import type { EvaluationRunClient } from "../../api/evaluationRunClient";
import {
  expectNoAccessibilityViolations,
  renderAtRoute,
} from "../../test/render";
import {
  EvaluationComparisonView,
  EvaluationRunInspection,
} from "./EvaluationRunInspection";

const run = parseEvaluationRunSummary(JSON.parse(summaryFixture));
const caseResult = parseEvaluationRunCase(JSON.parse(caseFixture));
const comparison = parseEvaluationComparison(JSON.parse(comparisonFixture));
const nonComparable = parseEvaluationComparison(
  JSON.parse(nonComparableFixture),
);

describe("evaluation run inspection", () => {
  it("shows frozen execution identity and expected-versus-actual case evidence", async () => {
    const user = userEvent.setup();
    const result = renderAtRoute(
      <EvaluationRunInspection client={new RunClientStub()} runId={run.id} />,
    );

    expect(
      await screen.findByRole("heading", { name: "Immutable run identity" }),
    ).toBeVisible();
    expect(screen.getByText(run.snapshotManifestSha256)).toBeVisible();
    expect(screen.getByText(run.agentBuild)).toBeVisible();
    expect(screen.getByText(run.chatModelIdentity)).toBeVisible();
    expect(screen.getByText(run.embeddingModelIdentity)).toBeVisible();
    await user.click(screen.getByRole("button", { name: "Case 1: Completed" }));

    expect(
      await screen.findByRole("heading", {
        name: "Expected and actual evidence",
      }),
    ).toBeVisible();
    expect(
      screen.getByRole("heading", { name: "Expected evidence", level: 4 }),
    ).toBeVisible();
    expect(
      screen.getByRole("heading", { name: "Actual evidence", level: 4 }),
    ).toBeVisible();
    expect(screen.getAllByText("Synthetic section 1")).toHaveLength(3);
    await expectNoAccessibilityViolations(result.container);
  });

  it("keeps quality deltas absent for non-comparable runs", async () => {
    const result = renderAtRoute(
      <EvaluationComparisonView
        client={new RunClientStub({ comparison: nonComparable })}
        leftRunId={nonComparable.leftRunId}
        rightRunId={nonComparable.rightRunId}
      />,
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "These runs are not comparable.",
    );
    expect(screen.getByText("Snapshot manifest SHA-256")).toBeVisible();
    expect(
      screen.queryByRole("table", { name: "Paired metric arithmetic" }),
    ).not.toBeInTheDocument();
    await expectNoAccessibilityViolations(result.container);
  });

  it("renders only a safe failure message when inspection cannot load", async () => {
    renderAtRoute(
      <EvaluationRunInspection
        client={new RunClientStub({ shouldFail: true })}
        runId={run.id}
      />,
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "The evaluation run could not be loaded.",
    );
    expect(
      screen.queryByText("provider diagnostic secret"),
    ).not.toBeInTheDocument();
  });

  it("clears and gates case inspection state when navigating to another run", async () => {
    const user = userEvent.setup();
    const secondRunId = "40000000-0000-4000-8000-000000000032";
    const firstCase = run.cases.at(0);
    if (firstCase === undefined)
      throw new Error("The run fixture has no case.");
    const secondRun = {
      ...run,
      id: secondRunId,
      cases: [{ ...firstCase }],
    };
    const client = new NavigationRunClient(secondRun);
    const result = renderAtRoute(
      <EvaluationRunInspection client={client} runId={run.id} />,
    );

    await user.click(
      await screen.findByRole("button", { name: "Case 1: Completed" }),
    );
    expect(await screen.findByText(caseResult.referenceAnswer)).toBeVisible();

    result.rerender(
      <EvaluationRunInspection client={client} runId={secondRunId} />,
    );

    expect(await screen.findByText(secondRunId)).toBeVisible();
    expect(
      screen.queryByText(caseResult.referenceAnswer),
    ).not.toBeInTheDocument();
  });

  it("does not restore a stale case when run navigation returns to an earlier identity", async () => {
    const user = userEvent.setup();
    const secondRunId = "40000000-0000-4000-8000-000000000032";
    const client = new DeferredRunClient();
    const result = renderAtRoute(
      <EvaluationRunInspection client={client} runId={run.id} />,
    );

    client.resolveRun(run.id, run);
    await user.click(
      await screen.findByRole("button", { name: "Case 1: Completed" }),
    );
    await waitFor(() => expect(client.caseRequestCount(run.id)).toBe(1));

    result.rerender(
      <EvaluationRunInspection client={client} runId={secondRunId} />,
    );
    await waitFor(() => expect(client.runRequestCount(secondRunId)).toBe(1));
    result.rerender(<EvaluationRunInspection client={client} runId={run.id} />);
    await waitFor(() => expect(client.runRequestCount(run.id)).toBe(1));

    client.resolveCase(run.id, caseResult);
    client.resolveRun(run.id, run);

    expect(await screen.findByText(run.id)).toBeVisible();
    expect(
      screen.queryByText(caseResult.referenceAnswer),
    ).not.toBeInTheDocument();
  });

  it("does not restore a stale comparison after an A-to-B-to-A identity race", async () => {
    const thirdRunId = "40000000-0000-4000-8000-000000000033";
    const client = new DeferredComparisonClient();
    const result = renderAtRoute(
      <EvaluationComparisonView
        client={client}
        leftRunId={comparison.leftRunId}
        rightRunId={comparison.rightRunId}
      />,
    );

    await waitFor(() =>
      expect(
        client.requestCount(comparison.leftRunId, comparison.rightRunId),
      ).toBe(1),
    );
    result.rerender(
      <EvaluationComparisonView
        client={client}
        leftRunId={comparison.rightRunId}
        rightRunId={thirdRunId}
      />,
    );
    await waitFor(() =>
      expect(client.requestCount(comparison.rightRunId, thirdRunId)).toBe(1),
    );
    result.rerender(
      <EvaluationComparisonView
        client={client}
        leftRunId={comparison.leftRunId}
        rightRunId={comparison.rightRunId}
      />,
    );
    await waitFor(() =>
      expect(
        client.requestCount(comparison.leftRunId, comparison.rightRunId),
      ).toBe(2),
    );

    client.resolve(
      comparison.leftRunId,
      comparison.rightRunId,
      staleComparison(),
    );
    expect(screen.getByText("Loading evaluation comparison...")).toBeVisible();

    client.resolve(comparison.leftRunId, comparison.rightRunId, comparison);
    expect(await screen.findByText(comparison.left.agentBuild)).toBeVisible();
    expect(screen.queryByText("stale-build")).not.toBeInTheDocument();
  });
});

class RunClientStub implements EvaluationRunClient {
  readonly #comparison: EvaluationComparison;
  readonly #shouldFail: boolean;

  constructor(
    options: {
      readonly comparison?: EvaluationComparison;
      readonly shouldFail?: boolean;
    } = {},
  ) {
    this.#comparison = options.comparison ?? comparison;
    this.#shouldFail = options.shouldFail ?? false;
  }

  getRun(runId: string, signal: AbortSignal): Promise<EvaluationRunSummary> {
    void runId;
    void signal;
    if (this.#shouldFail)
      return Promise.reject(new Error("provider diagnostic secret"));
    return Promise.resolve(run);
  }

  getCase(
    runId: string,
    caseId: string,
    signal: AbortSignal,
  ): Promise<EvaluationRunCase> {
    void runId;
    void caseId;
    void signal;
    return Promise.resolve(caseResult);
  }

  compareRuns(
    leftRunId: string,
    rightRunId: string,
    signal: AbortSignal,
  ): Promise<EvaluationComparison> {
    void leftRunId;
    void rightRunId;
    void signal;
    return Promise.resolve(this.#comparison);
  }
}

class NavigationRunClient extends RunClientStub {
  readonly #secondRun: EvaluationRunSummary;

  constructor(secondRun: EvaluationRunSummary) {
    super();
    this.#secondRun = secondRun;
  }

  override getRun(
    runId: string,
    signal: AbortSignal,
  ): Promise<EvaluationRunSummary> {
    void signal;
    return Promise.resolve(
      runId === this.#secondRun.id ? this.#secondRun : run,
    );
  }

  override getCase(
    runId: string,
    caseId: string,
    signal: AbortSignal,
  ): Promise<EvaluationRunCase> {
    void caseId;
    void signal;
    if (runId === this.#secondRun.id) {
      return new Promise(() => undefined);
    }
    return Promise.resolve(caseResult);
  }
}

class DeferredRunClient extends RunClientStub {
  readonly #runs = new Map<string, Deferred<EvaluationRunSummary>[]>();
  readonly #cases = new Map<string, Deferred<EvaluationRunCase>[]>();

  override getRun(
    runId: string,
    signal: AbortSignal,
  ): Promise<EvaluationRunSummary> {
    void signal;
    const request = deferred<EvaluationRunSummary>();
    this.#requestsFor(this.#runs, runId).push(request);
    return request.promise;
  }

  override getCase(
    runId: string,
    caseId: string,
    signal: AbortSignal,
  ): Promise<EvaluationRunCase> {
    void caseId;
    void signal;
    const request = deferred<EvaluationRunCase>();
    this.#requestsFor(this.#cases, runId).push(request);
    return request.promise;
  }

  resolveRun(runId: string, value: EvaluationRunSummary): void {
    this.#takeRequest(this.#runs, runId).resolve(value);
  }

  resolveCase(runId: string, value: EvaluationRunCase): void {
    this.#takeRequest(this.#cases, runId).resolve(value);
  }

  runRequestCount(runId: string): number {
    return this.#runs.get(runId)?.length ?? 0;
  }

  caseRequestCount(runId: string): number {
    return this.#cases.get(runId)?.length ?? 0;
  }

  #requestsFor<T>(
    requests: Map<string, Deferred<T>[]>,
    identity: string,
  ): Deferred<T>[] {
    const existing = requests.get(identity);
    if (existing !== undefined) return existing;
    const created: Deferred<T>[] = [];
    requests.set(identity, created);
    return created;
  }

  #takeRequest<T>(
    requests: Map<string, Deferred<T>[]>,
    identity: string,
  ): Deferred<T> {
    const request = requests.get(identity)?.shift();
    if (request === undefined)
      throw new Error(`No pending request for ${identity}.`);
    return request;
  }
}

class DeferredComparisonClient extends RunClientStub {
  readonly #requests = new Map<string, Deferred<EvaluationComparison>[]>();

  override compareRuns(
    leftRunId: string,
    rightRunId: string,
    signal: AbortSignal,
  ): Promise<EvaluationComparison> {
    void signal;
    const request = deferred<EvaluationComparison>();
    const identity = comparisonIdentity(leftRunId, rightRunId);
    const existing = this.#requests.get(identity);
    if (existing === undefined) {
      this.#requests.set(identity, [request]);
    } else {
      existing.push(request);
    }
    return request.promise;
  }

  requestCount(leftRunId: string, rightRunId: string): number {
    return (
      this.#requests.get(comparisonIdentity(leftRunId, rightRunId))?.length ?? 0
    );
  }

  resolve(
    leftRunId: string,
    rightRunId: string,
    value: EvaluationComparison,
  ): void {
    const request = this.#requests
      .get(comparisonIdentity(leftRunId, rightRunId))
      ?.shift();
    if (request === undefined) {
      throw new Error("No pending comparison request.");
    }
    request.resolve(value);
  }
}

function staleComparison(): EvaluationComparison {
  return {
    ...comparison,
    left: { ...comparison.left, agentBuild: "stale-build" },
  };
}

function comparisonIdentity(leftRunId: string, rightRunId: string): string {
  return `${leftRunId}:${rightRunId}`;
}

interface Deferred<T> {
  readonly promise: Promise<T>;
  resolve(value: T): void;
}

function deferred<T>(): Deferred<T> {
  let resolvePromise: (value: T) => void = () => undefined;
  const promise = new Promise<T>((resolve) => {
    resolvePromise = resolve;
  });
  return { promise, resolve: resolvePromise };
}
