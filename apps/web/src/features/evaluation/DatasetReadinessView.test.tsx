import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import catalogFixture from "../../../../../contracts/evaluation/v1/fixtures/dataset-catalog-response.json?raw";
import detailFixture from "../../../../../contracts/evaluation/v1/fixtures/dataset-detail-response.json?raw";
import {
  parseEvaluationDatasetCatalog,
  parseEvaluationDatasetDetail,
  type EvaluationDatasetCatalogEntry,
  type EvaluationDatasetDetail,
  type EvaluationDatasetPreflightRequest,
  type EvaluationDatasetPreflightResponse,
} from "../../api/evaluationCatalog";
import {
  EvaluationCatalogRequestError,
  type EvaluationCatalogClient,
} from "../../api/evaluationCatalogClient";
import {
  expectNoAccessibilityViolations,
  renderAtRoute,
} from "../../test/render";
import {
  DatasetReadinessView,
  type EvaluationSnapshotOption,
} from "./DatasetReadinessView";

const dataset = requiredDataset(
  parseEvaluationDatasetCatalog(JSON.parse(catalogFixture)),
);
const detail = parseEvaluationDatasetDetail(JSON.parse(detailFixture));

const compatibleSnapshot: EvaluationSnapshotOption = {
  corpusId: detail.revision.corpusId,
  snapshotId: "20000000-0000-4000-8000-000000000021",
  snapshotManifestSha256: "a".repeat(64),
  label: "Published dataset snapshot",
};
const incompatibleSnapshot: EvaluationSnapshotOption = {
  corpusId: detail.revision.corpusId,
  snapshotId: "20000000-0000-4000-8000-000000000022",
  snapshotManifestSha256: "b".repeat(64),
  label: "Published snapshot missing a required source",
};
const foreignCorpusSnapshot: EvaluationSnapshotOption = {
  corpusId: "10000000-0000-4000-8000-000000000023",
  snapshotId: "20000000-0000-4000-8000-000000000023",
  snapshotManifestSha256: "c".repeat(64),
  label: "Foreign corpus snapshot",
};
const secondDataset: EvaluationDatasetCatalogEntry = {
  ...dataset,
  revision: {
    ...dataset.revision,
    id: "10000000-0000-4000-8000-000000000024",
    corpusId: foreignCorpusSnapshot.corpusId,
  },
};
const secondDetail: EvaluationDatasetDetail = {
  ...detail,
  revision: secondDataset.revision,
};

describe("dataset readiness view", () => {
  it("shows immutable identity, source and review state, and the technical notice", async () => {
    const client = new DatasetClientStub();
    const result = renderAtRoute(
      <DatasetReadinessView
        client={client}
        snapshotOptions={[compatibleSnapshot]}
        onStartRun={vi.fn()}
      />,
    );
    const user = userEvent.setup();

    await user.selectOptions(
      await screen.findByRole("combobox", { name: "Dataset revision" }),
      dataset.revision.id,
    );

    expect(
      await screen.findByRole("heading", {
        name: "Immutable dataset identity",
      }),
    ).toBeVisible();
    expect(screen.getByText(detail.revision.manifestSha256)).toBeVisible();
    expect(screen.getByText(/Approved - Available/)).toBeVisible();
    expect(screen.getByText("Bound to corpus")).toBeVisible();
    expect(screen.getByRole("note")).toHaveTextContent(
      "technical measurement of corpus-grounded behavior",
    );
    await expectNoAccessibilityViolations(result.container);
  });

  it("shows missing requirements and cannot start an incompatible selection", async () => {
    const onStartRun = vi.fn();
    const client = new DatasetClientStub({ incompatible: true });
    const user = userEvent.setup();
    renderAtRoute(
      <DatasetReadinessView
        client={client}
        snapshotOptions={[incompatibleSnapshot]}
        onStartRun={onStartRun}
      />,
    );

    await user.selectOptions(
      await screen.findByRole("combobox", { name: "Dataset revision" }),
      dataset.revision.id,
    );
    await screen.findByRole("heading", { name: "Immutable dataset identity" });
    await user.selectOptions(
      screen.getByRole("combobox", { name: "Corpus and immutable snapshot" }),
      `${incompatibleSnapshot.corpusId}:${incompatibleSnapshot.snapshotId}`,
    );
    await user.click(
      screen.getByRole("button", { name: "Check compatibility" }),
    );

    const incompatibilityAlert = await screen.findByRole("alert");
    expect(incompatibilityAlert).toHaveTextContent(
      "No evaluation run can be started.",
    );
    expect(
      within(incompatibilityAlert).getByText("official-source"),
    ).toBeVisible();
    expect(
      screen.getByRole("button", { name: "Start evaluation run" }),
    ).toBeDisabled();
    expect(onStartRun).not.toHaveBeenCalled();
    expect(client.preflightRequests).toEqual([
      {
        datasetRevisionId: detail.revision.id,
        corpusId: incompatibleSnapshot.corpusId,
        snapshotId: incompatibleSnapshot.snapshotId,
      },
    ]);
  });

  it("permits a run only after a compatible preflight for the selected identities", async () => {
    const onStartRun = vi.fn();
    const client = new DatasetClientStub();
    const user = userEvent.setup();
    renderAtRoute(
      <DatasetReadinessView
        client={client}
        snapshotOptions={[compatibleSnapshot]}
        onStartRun={onStartRun}
      />,
    );

    await user.selectOptions(
      await screen.findByRole("combobox", { name: "Dataset revision" }),
      dataset.revision.id,
    );
    await screen.findByRole("heading", { name: "Immutable dataset identity" });
    await user.selectOptions(
      screen.getByRole("combobox", { name: "Corpus and immutable snapshot" }),
      `${compatibleSnapshot.corpusId}:${compatibleSnapshot.snapshotId}`,
    );
    expect(
      screen.getByRole("heading", {
        name: "Selected immutable snapshot identity",
      }),
    ).toBeVisible();
    expect(
      screen.getByText(compatibleSnapshot.snapshotManifestSha256),
    ).toBeVisible();
    await user.click(
      screen.getByRole("button", { name: "Check compatibility" }),
    );

    expect(await screen.findByRole("status")).toHaveTextContent(
      "compatible with the selected immutable snapshot",
    );
    await user.click(
      screen.getByRole("button", { name: "Start evaluation run" }),
    );
    expect(onStartRun).toHaveBeenCalledWith({
      datasetRevisionId: detail.revision.id,
      corpusId: compatibleSnapshot.corpusId,
      snapshotId: compatibleSnapshot.snapshotId,
      compatible: true,
      missingRequirements: [],
    });
  });

  it("only offers snapshots from the selected dataset corpus", async () => {
    const client = new DatasetClientStub();
    const user = userEvent.setup();
    renderAtRoute(
      <DatasetReadinessView
        client={client}
        snapshotOptions={[foreignCorpusSnapshot, compatibleSnapshot]}
        onStartRun={vi.fn()}
      />,
    );

    await user.selectOptions(
      await screen.findByRole("combobox", { name: "Dataset revision" }),
      dataset.revision.id,
    );
    const snapshotSelect = await screen.findByRole("combobox", {
      name: "Corpus and immutable snapshot",
    });

    expect(
      within(snapshotSelect).queryByRole("option", {
        name: foreignCorpusSnapshot.label,
      }),
    ).not.toBeInTheDocument();
    await user.selectOptions(
      snapshotSelect,
      `${compatibleSnapshot.corpusId}:${compatibleSnapshot.snapshotId}`,
    );
    await user.click(
      screen.getByRole("button", { name: "Check compatibility" }),
    );

    expect(client.preflightRequests).toEqual([
      {
        datasetRevisionId: detail.revision.id,
        corpusId: detail.revision.corpusId,
        snapshotId: compatibleSnapshot.snapshotId,
      },
    ]);
  });

  it("shows an operational preflight failure instead of incompatibility", async () => {
    const client = new DatasetClientStub({
      preflightError: new EvaluationCatalogRequestError(
        "not_found",
        "The evaluation dataset was not found.",
        "50000000-0000-4000-8000-000000000022",
        undefined,
      ),
    });
    const user = userEvent.setup();
    renderAtRoute(
      <DatasetReadinessView
        client={client}
        snapshotOptions={[compatibleSnapshot]}
        onStartRun={vi.fn()}
      />,
    );

    await selectDatasetAndSnapshot(user, compatibleSnapshot);
    await user.click(
      screen.getByRole("button", { name: "Check compatibility" }),
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Compatibility could not be checked.",
    );
    expect(
      screen.queryByText("This selection is incompatible."),
    ).not.toBeInTheDocument();
  });

  it("clears a selected snapshot when the dataset revision changes", async () => {
    const client = new DatasetClientStub({
      datasets: [dataset, secondDataset],
      details: new Map([
        [dataset.revision.id, detail],
        [secondDataset.revision.id, secondDetail],
      ]),
    });
    const user = userEvent.setup();
    renderAtRoute(
      <DatasetReadinessView
        client={client}
        snapshotOptions={[compatibleSnapshot, foreignCorpusSnapshot]}
        onStartRun={vi.fn()}
      />,
    );

    await selectDatasetAndSnapshot(user, compatibleSnapshot);
    await user.selectOptions(
      screen.getByRole("combobox", { name: "Dataset revision" }),
      secondDataset.revision.id,
    );
    await screen.findByText(secondDetail.revision.manifestSha256);

    expect(
      screen.getByRole("combobox", { name: "Corpus and immutable snapshot" }),
    ).toHaveValue("");
  });

  it("aborts an in-flight preflight when unmounted", async () => {
    const client = new DatasetClientStub({ pendingPreflight: true });
    const user = userEvent.setup();
    const result = renderAtRoute(
      <DatasetReadinessView
        client={client}
        snapshotOptions={[compatibleSnapshot]}
        onStartRun={vi.fn()}
      />,
    );

    await selectDatasetAndSnapshot(user, compatibleSnapshot);
    await user.click(
      screen.getByRole("button", { name: "Check compatibility" }),
    );
    result.unmount();

    expect(client.preflightSignal?.aborted).toBe(true);
  });
});

class DatasetClientStub implements EvaluationCatalogClient {
  readonly preflightRequests: EvaluationDatasetPreflightRequest[] = [];
  preflightSignal: AbortSignal | undefined;

  constructor(
    private readonly options: {
      readonly datasets?: readonly EvaluationDatasetCatalogEntry[];
      readonly details?: ReadonlyMap<string, EvaluationDatasetDetail>;
      readonly incompatible?: boolean;
      readonly pendingPreflight?: boolean;
      readonly preflightError?: Error;
    } = {},
  ) {}

  listDatasets(
    _signal: AbortSignal,
  ): Promise<readonly EvaluationDatasetCatalogEntry[]> {
    void _signal;
    return Promise.resolve(this.options.datasets ?? [dataset]);
  }

  getDataset(
    _datasetRevisionId: string,
    _signal: AbortSignal,
  ): Promise<EvaluationDatasetDetail> {
    void _signal;
    return Promise.resolve(
      this.options.details?.get(_datasetRevisionId) ?? detail,
    );
  }

  preflightDataset(
    request: EvaluationDatasetPreflightRequest,
    _signal: AbortSignal,
  ): Promise<EvaluationDatasetPreflightResponse> {
    this.preflightSignal = _signal;
    this.preflightRequests.push(request);
    if (this.options.pendingPreflight) {
      return new Promise(() => undefined);
    }
    if (this.options.preflightError !== undefined) {
      return Promise.reject(this.options.preflightError);
    }
    if (this.options.incompatible) {
      return Promise.reject(
        new EvaluationCatalogRequestError(
          "snapshot_incompatible",
          "The selected snapshot is missing a required source.",
          "40000000-0000-4000-8000-000000000021",
          [
            {
              sourceAlias: "official-source",
              locator: "Article 1",
              reason:
                "The required source is not a member of the selected snapshot.",
            },
          ],
        ),
      );
    }
    return Promise.resolve({
      ...request,
      compatible: true,
      missingRequirements: [],
    });
  }
}

async function selectDatasetAndSnapshot(
  user: ReturnType<typeof userEvent.setup>,
  snapshot: EvaluationSnapshotOption,
): Promise<void> {
  await user.selectOptions(
    await screen.findByRole("combobox", { name: "Dataset revision" }),
    dataset.revision.id,
  );
  await screen.findByRole("heading", { name: "Immutable dataset identity" });
  await user.selectOptions(
    screen.getByRole("combobox", { name: "Corpus and immutable snapshot" }),
    `${snapshot.corpusId}:${snapshot.snapshotId}`,
  );
}

function requiredDataset(
  catalog: readonly EvaluationDatasetCatalogEntry[],
): EvaluationDatasetCatalogEntry {
  const firstDataset = catalog[0];
  if (firstDataset === undefined) {
    throw new Error("Evaluation dataset catalog fixture is empty.");
  }
  return firstDataset;
}
