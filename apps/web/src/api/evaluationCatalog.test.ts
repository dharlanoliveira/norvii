import { describe, expect, it } from "vitest";

import catalogFixture from "../../../../contracts/evaluation/v1/fixtures/dataset-catalog-response.json?raw";
import detailFixture from "../../../../contracts/evaluation/v1/fixtures/dataset-detail-response.json?raw";
import notFoundFixture from "../../../../contracts/evaluation/v1/fixtures/dataset-preflight-error-not-found.json?raw";
import incompatibleFixture from "../../../../contracts/evaluation/v1/fixtures/dataset-preflight-error-snapshot-incompatible.json?raw";
import preflightFixture from "../../../../contracts/evaluation/v1/fixtures/dataset-preflight-success.json?raw";

import {
  assertEvaluationDatasetPreflightRequest,
  parseEvaluationCatalogErrorEnvelope,
  parseEvaluationDatasetCatalog,
  parseEvaluationDatasetDetail,
  parseEvaluationDatasetPreflightResponse,
} from "./evaluationCatalog";

describe("evaluation dataset catalog contract", () => {
  it("parses immutable catalog and detail projections", () => {
    const catalog = parseEvaluationDatasetCatalog(JSON.parse(catalogFixture));
    const detail = parseEvaluationDatasetDetail(JSON.parse(detailFixture));

    expect(catalog).toHaveLength(1);
    expect(catalog[0]).toMatchObject({
      available: true,
      revision: {
        datasetKey: "synthetic-evaluation-dataset",
        queryLanguages: ["en", "pt"],
      },
    });
    expect(detail.sources[0]).toMatchObject({
      sourceAlias: "official-source",
      bound: true,
    });
    expect(detail.starters[0]).toMatchObject({
      rank: 1,
      reviewEligible: true,
    });
  });

  it("accepts an unavailable revision only when its review is unavailable", () => {
    const catalog = JSON.parse(catalogFixture) as Array<
      Record<string, unknown>
    >;
    const catalogEntry = catalog[0];
    if (catalogEntry === undefined)
      throw new Error("Catalog fixture is empty.");
    const review = catalogEntry.review as Record<string, unknown>;

    const unavailable = parseEvaluationDatasetCatalog([
      {
        ...catalogEntry,
        available: false,
        review: {
          ...review,
          decision: "pending",
          publicationState: "draft",
        },
      },
    ]);

    expect(unavailable[0]).toMatchObject({
      available: false,
      review: { decision: "pending", publicationState: "draft" },
    });
  });

  it("parses the immutable compatible preflight selection", () => {
    const response = parseEvaluationDatasetPreflightResponse(
      JSON.parse(preflightFixture),
    );

    expect(response.compatible).toBe(true);
    expect(response.missingRequirements).toEqual([]);
  });

  it("preserves bounded incompatibility requirements and allows their absence", () => {
    const incompatible = parseEvaluationCatalogErrorEnvelope(
      JSON.parse(incompatibleFixture),
    );
    const notFound = parseEvaluationCatalogErrorEnvelope(
      JSON.parse(notFoundFixture),
    );

    expect(incompatible.error.missingRequirements).toEqual([
      {
        sourceAlias: "official-source",
        locator: "Article 1",
        reason: "The required source is not a member of the selected snapshot.",
      },
    ]);
    expect(notFound.error.missingRequirements).toBeUndefined();
  });

  it("rejects unknown fields, invalid availability, and malformed identities", () => {
    const catalog = JSON.parse(catalogFixture) as Array<
      Record<string, unknown>
    >;
    const catalogEntry = catalog[0];
    if (catalogEntry === undefined)
      throw new Error("Catalog fixture is empty.");
    const revision = catalogEntry.revision as Record<string, unknown>;

    expect(() =>
      parseEvaluationDatasetCatalog([
        { ...catalogEntry, revision: { ...revision, leaked: true } },
      ]),
    ).toThrow("unsupported field");
    expect(() =>
      parseEvaluationDatasetCatalog([{ ...catalogEntry, available: false }]),
    ).toThrow("availability must match");
    expect(() =>
      assertEvaluationDatasetPreflightRequest({
        datasetRevisionId: "invalid",
        corpusId: "10000000-0000-4000-8000-000000000021",
        snapshotId: "20000000-0000-4000-8000-000000000021",
      }),
    ).toThrow("UUID");
  });

  it("rejects oversized or malformed compatibility diagnostics", () => {
    const error = JSON.parse(incompatibleFixture) as {
      error: Record<string, unknown>;
    };
    const requirements = error.error.missingRequirements as Array<unknown>;
    const requirement = requirements[0] as Record<string, unknown> | undefined;
    if (requirement === undefined) {
      throw new Error("Incompatible fixture has no missing requirement.");
    }

    expect(() =>
      parseEvaluationCatalogErrorEnvelope({
        error: {
          ...error.error,
          missingRequirements: Array.from({ length: 33 }, () => requirement),
        },
      }),
    ).toThrow("cannot exceed 32");
    expect(() =>
      parseEvaluationCatalogErrorEnvelope({
        error: {
          ...error.error,
          missingRequirements: [{ ...requirement, privateDetail: "no" }],
        },
      }),
    ).toThrow("unsupported field");
  });
});
