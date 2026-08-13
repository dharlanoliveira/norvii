import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach, beforeEach } from "vitest";

import { i18n } from "../i18n/config";

class ResizeObserverMock implements ResizeObserver {
  readonly observedElements = new Set<Element>();

  observe(target: Element): void {
    this.observedElements.add(target);
  }

  unobserve(target: Element): void {
    this.observedElements.delete(target);
  }

  disconnect(): void {
    this.observedElements.clear();
  }
}

globalThis.ResizeObserver = ResizeObserverMock;
Element.prototype.scrollTo = (): void => undefined;

beforeEach(async () => {
  await i18n.changeLanguage("en");
  window.history.replaceState({}, "", "/");
});

afterEach(() => {
  cleanup();
});
