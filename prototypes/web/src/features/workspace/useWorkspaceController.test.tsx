import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { corpora } from "../../fixtures/legal-content/corpora";
import { useWorkspaceController } from "./useWorkspaceController";

describe("useWorkspaceController", () => {
  it("preserves the active source while switching modes", () => {
    const corpus = corpora.at(0);
    if (!corpus) {
      throw new Error("Expected the Brazilian corpus fixture to be present.");
    }
    const { result } = renderHook(() => useWorkspaceController(corpus));

    act(() => result.current.selectSource("lgpd-law"));
    expect(result.current.mode).toBe("source");
    expect(result.current.viewer).toEqual({
      sourceId: "lgpd-law",
      sectionId: "article-6",
    });

    act(() => result.current.selectMode("chat"));
    expect(result.current.viewer.sourceId).toBe("lgpd-law");
  });

  it("opens only citations belonging to the active corpus", () => {
    const corpus = corpora.at(0);
    if (!corpus) {
      throw new Error("Expected the Brazilian corpus fixture to be present.");
    }
    const { result } = renderHook(() => useWorkspaceController(corpus));

    act(() =>
      result.current.openCitation({
        id: "valid",
        sourceId: "lgpd-law",
        sectionId: "article-18",
        label: "LGPD, Art. 18",
      }),
    );
    expect(result.current.viewer.sectionId).toBe("article-18");

    act(() =>
      result.current.openCitation({
        id: "foreign",
        sourceId: "gdpr-regulation",
        sectionId: "article-15",
        label: "GDPR, Article 15",
      }),
    );
    expect(result.current.viewer.sourceId).toBe("lgpd-law");
  });
});
