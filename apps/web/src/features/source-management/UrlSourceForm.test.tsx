import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { renderAtRoute } from "../../test/render";
import { UrlSourceForm } from "./UrlSourceForm";

describe("URL source form", () => {
  it("submits a valid HTTPS origin and communicates queueing", async () => {
    const user = userEvent.setup();
    const submit = vi.fn().mockResolvedValue(undefined);
    renderAtRoute(<UrlSourceForm onSubmit={submit} />);

    await user.type(
      screen.getByRole("textbox", { name: "Source title" }),
      "Official law",
    );
    await user.type(
      screen.getByRole("textbox", { name: "Official HTTPS URL" }),
      "https://example.org/law",
    );
    await user.click(screen.getByRole("button", { name: "Add URL source" }));

    expect(await screen.findByRole("status")).toHaveTextContent(
      "Source queued for ingestion.",
    );
    expect(submit).toHaveBeenCalledWith(
      { title: "Official law", url: "https://example.org/law" },
      expect.any(AbortSignal),
    );
  });

  it("uses browser validation to reject a non-HTTPS URL", async () => {
    const user = userEvent.setup();
    const submit = vi.fn().mockResolvedValue(undefined);
    renderAtRoute(<UrlSourceForm onSubmit={submit} />);

    await user.type(
      screen.getByRole("textbox", { name: "Source title" }),
      "Unsafe law",
    );
    await user.type(
      screen.getByRole("textbox", { name: "Official HTTPS URL" }),
      "http://example.org",
    );
    await user.click(screen.getByRole("button", { name: "Add URL source" }));

    expect(submit).not.toHaveBeenCalled();
  });
});
