import { render, type RenderOptions } from "@testing-library/react";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router-dom";

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
