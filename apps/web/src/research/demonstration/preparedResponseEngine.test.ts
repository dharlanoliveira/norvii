import { describe, expect, it } from "vitest";

import { createDemonstrationCatalog } from "./createDemonstrationCatalog";
import { createPreparedResponseEngine } from "./preparedResponseEngine";

describe("prepared response engine", () => {
  it("keeps answered, abstained, and failed outcomes inside the active corpus", () => {
    const catalog = createDemonstrationCatalog();
    const corpus = catalog.findCorpus("eu-data-protection");
    const brazilCorpus = catalog.findCorpus("brazil-data-protection");
    const engine = createPreparedResponseEngine();

    if (!corpus || !brazilCorpus) {
      throw new Error("Expected demonstration corpora.");
    }

    const answered = engine.resolve(
      corpus,
      "What principles govern personal data processing?",
    );
    const abstained = engine.resolve(corpus, "Who won the latest election?");
    const failed = engine.resolve(corpus, "Please simulate failure");
    const isolated = engine.resolve(brazilCorpus, "controller responsibility");

    expect(answered.status).toBe("answered");
    expect(answered.parts).toContainEqual({
      type: "citation",
      citation: {
        id: "gdpr-article-5",
        sourceId: "gdpr",
        locationId: "article-5",
        label: "GDPR, Article 5",
      },
    });
    expect(abstained).toEqual({ status: "abstained", parts: [] });
    expect(failed).toEqual({
      status: "failed",
      code: "prepared-response-failed",
      parts: [],
    });
    expect(isolated).toEqual({ status: "abstained", parts: [] });
  });
});
