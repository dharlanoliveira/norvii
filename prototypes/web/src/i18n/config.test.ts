import { describe, expect, it } from "vitest";

import { englishTranslation } from "./en/translation";
import { portugueseTranslation } from "./pt/translation";

function collectKeys(value: object, prefix = ""): string[] {
  const entries = Object.entries(value) as [string, unknown][];

  return entries.flatMap(([key, entry]) => {
    const path = prefix ? `${prefix}.${key}` : key;
    return typeof entry === "object" && entry !== null
      ? collectKeys(entry, path)
      : [path];
  });
}

describe("localization resources", () => {
  it("keeps English and Portuguese keys structurally identical", () => {
    expect(collectKeys(portugueseTranslation)).toEqual(
      collectKeys(englishTranslation),
    );
  });
});
