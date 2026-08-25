import { describe, expect, it } from "vitest";

import { i18n } from "./config";
import { englishTranslation } from "./en/translation";
import { portugueseTranslation } from "./pt/translation";

function collectKeys(value: object, prefix = ""): string[] {
  return Object.entries(value).flatMap(([key, child]) => {
    const path = prefix ? `${prefix}.${key}` : key;
    return typeof child === "object" && child !== null
      ? collectKeys(child as object, path)
      : [path];
  });
}

describe("interface localization", () => {
  it("starts in English with structurally complete Portuguese resources", () => {
    expect(i18n.language).toBe("en");
    expect(collectKeys(portugueseTranslation).sort()).toEqual(
      collectKeys(englishTranslation).sort(),
    );
  });

  it("provides localized planned-hybrid research record labels", () => {
    for (const translation of [englishTranslation, portugueseTranslation]) {
      expect(typeof translation.chat.retrievalStage.vector).toBe("string");
      expect(typeof translation.chat.retrievalStage.planning).toBe("string");
      expect(typeof translation.chat.retrievalStage.graph).toBe("string");
      expect(typeof translation.chat.retrievalStageReason.not_relevant).toBe(
        "string",
      );
      expect(
        typeof translation.chat.retrievalStageReason.graph_release_unavailable,
      ).toBe("string");
      expect(
        typeof translation.chat.retrievalStageReason.graph_unavailable,
      ).toBe("string");
      expect(
        typeof translation.chat.retrievalStageReason.planner_unavailable,
      ).toBe("string");
      expect(typeof translation.chat.evidenceContribution.vector).toBe(
        "string",
      );
      expect(typeof translation.chat.evidenceContribution.graph).toBe("string");
      expect(
        typeof translation.chat.evidenceContribution.vector_and_graph,
      ).toBe("string");
    }
  });
});
