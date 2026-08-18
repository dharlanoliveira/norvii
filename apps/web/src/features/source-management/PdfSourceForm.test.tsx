import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { renderAtRoute } from "../../test/render";
import { ResearchRequestError } from "../../api/researchProvider";
import { PdfSourceForm } from "./PdfSourceForm";

describe("PDF source form", () => {
  it("uploads a valid PDF with accessible progress and queue outcome", async () => {
    const user = userEvent.setup();
    let complete: (() => void) | undefined;
    const submit = vi.fn().mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          complete = resolve;
        }),
    );
    renderAtRoute(<PdfSourceForm onSubmit={submit} />);
    const file = new File(["%PDF-generated-test"], "official.pdf", {
      type: "application/pdf",
    });

    await user.type(
      screen.getByRole("textbox", { name: "Source title" }),
      "Official PDF",
    );
    await user.upload(screen.getByLabelText("PDF document"), file);
    await user.click(screen.getByRole("button", { name: "Upload PDF source" }));

    expect(
      screen.getByRole("progressbar", { name: "Uploading PDF" }),
    ).toBeVisible();
    complete?.();
    expect(await screen.findByRole("status")).toHaveTextContent(
      "Source queued for ingestion.",
    );
    expect(submit).toHaveBeenCalledWith(
      "Official PDF",
      file,
      expect.any(AbortSignal),
    );
  });

  it("rejects a non-PDF file before upload", async () => {
    const user = userEvent.setup({ applyAccept: false });
    const submit = vi.fn().mockResolvedValue(undefined);
    renderAtRoute(<PdfSourceForm onSubmit={submit} />);

    await user.type(
      screen.getByRole("textbox", { name: "Source title" }),
      "Text file",
    );
    await user.upload(
      screen.getByLabelText("PDF document"),
      new File(["plain text"], "notes.txt", { type: "text/plain" }),
    );
    await user.click(screen.getByRole("button", { name: "Upload PDF source" }));

    expect(await screen.findByRole("alert")).toBeVisible();
    expect(submit).not.toHaveBeenCalled();
  });

  it("distinguishes a duplicate PDF from an invalid file", async () => {
    const user = userEvent.setup();
    const submit = vi
      .fn()
      .mockRejectedValue(
        new ResearchRequestError(
          "duplicate_source",
          "Duplicate source.",
          undefined,
          "40000000-0000-4000-8000-000000000001",
        ),
      );
    renderAtRoute(<PdfSourceForm onSubmit={submit} />);

    await user.type(
      screen.getByRole("textbox", { name: "Source title" }),
      "Duplicate PDF",
    );
    await user.upload(
      screen.getByLabelText("PDF document"),
      new File(["%PDF-generated-test"], "official.pdf", {
        type: "application/pdf",
      }),
    );
    await user.click(screen.getByRole("button", { name: "Upload PDF source" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "This source is already registered in this corpus.",
    );
  });
});
