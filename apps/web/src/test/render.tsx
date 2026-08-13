import axe from "axe-core";
import { render, type RenderOptions } from "@testing-library/react";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router-dom";
import { expect } from "vitest";

export function renderAtRoute(
  element: ReactElement,
  route = "/",
  options?: Omit<RenderOptions, "wrapper">,
) {
  return render(element, {
    wrapper: ({ children }) => (
      <MemoryRouter initialEntries={[route]}>{children}</MemoryRouter>
    ),
    ...options,
  });
}

export async function expectNoAccessibilityViolations(
  container: Element,
): Promise<void> {
  const result = await axe.run(container, {
    rules: { "color-contrast": { enabled: false } },
  });
  expect(result.violations).toEqual([]);
}
