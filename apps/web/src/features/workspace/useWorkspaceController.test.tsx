import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { createDemonstrationCatalog } from "../../research/demonstration/createDemonstrationCatalog";
import { useWorkspaceController } from "./useWorkspaceController";

describe("workspace controller", () => {
  it("preserves source locations while source selection and mode change", () => {
    const corpus =
      createDemonstrationCatalog().findCorpus("eu-data-protection");
    if (!corpus) throw new Error("Expected demonstration corpus.");

    const { result } = renderHook(() => useWorkspaceController(corpus));

    act(() => result.current.selectSource("gdpr"));
    expect(result.current.mode).toBe("source");
    expect(result.current.selectedSource?.id).toBe("gdpr");
    expect(result.current.activeLocationId).toBe("article-5");

    act(() => result.current.changeLocation("article-24"));
    act(() => result.current.setMode("chat"));
    act(() => result.current.selectSource("edpb-controller-guidelines"));
    act(() => result.current.selectSource("gdpr"));

    expect(result.current.activeLocationId).toBe("article-24");
  });
});
