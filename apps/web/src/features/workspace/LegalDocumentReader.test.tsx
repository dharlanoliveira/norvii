import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { DocumentResponse } from "../../api/contract";
import { renderAtRoute } from "../../test/render";
import { LegalDocumentReader } from "./LegalDocumentReader";

describe("legal document reader", () => {
  it("renders structured locations as readable sections and navigates with one control", async () => {
    const onSelect = vi.fn();
    const user = userEvent.setup();

    renderAtRoute(
      <LegalDocumentReader
        document={document()}
        selectedUnitId="unit-title"
        onSelect={onSelect}
      />,
    );

    expect(
      screen.getByRole("article", { name: "Preamble and recitals" }),
    ).toBeVisible();
    expect(screen.getByRole("article", { name: "Article 1" })).toBeVisible();
    expect(screen.getByText("Purpose and scope.")).toBeVisible();
    expect(
      screen.queryByRole("button", { name: "Article 1" }),
    ).not.toBeInTheDocument();

    await user.selectOptions(
      screen.getByRole("combobox", { name: "Document location" }),
      "unit-article",
    );

    expect(onSelect).toHaveBeenCalledWith("unit-article");
  });

  it("keeps Brazilian paragraphs and items inside their article context", () => {
    renderAtRoute(
      <LegalDocumentReader
        document={brazilianDocument()}
        selectedUnitId="article-3"
        onSelect={vi.fn()}
      />,
    );

    const article = screen.getByRole("article", { name: "Art. 3\u00ba" });
    expect(article).toHaveTextContent(
      "I - a opera\u00e7\u00e3o seja realizada no territ\u00f3rio nacional",
    );
    expect(article).toHaveTextContent(
      "\u00a7 1\u00ba Consideram-se coletados no territ\u00f3rio nacional",
    );
    expect(
      screen.queryByRole("article", { name: "\u00a7 1\u00ba" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("option", { name: "\u00a7 1\u00ba" }),
    ).not.toBeInTheDocument();
  });

  it("does not duplicate articles inside their structural ancestors", () => {
    renderAtRoute(
      <LegalDocumentReader
        document={hierarchicalDocument()}
        selectedUnitId="chapter-one"
        onSelect={vi.fn()}
      />,
    );

    expect(
      screen.getByRole("article", { name: "Chapter I General provisions" }),
    ).not.toHaveTextContent("Article caput");
    expect(
      screen.getByRole("article", { name: "Art. 1\u00ba" }),
    ).toHaveTextContent("I - first item");
  });
});

function document(): DocumentResponse {
  const text = "Preamble introduction.\nArticle 1\nPurpose and scope.";
  return {
    id: "50000000-0000-4000-8000-000000000001",
    sourceRevisionId: "40000000-0000-4000-8000-000000000001",
    pipelineVersion: "corpus-ingestion-v1",
    text,
    textSha256: "a".repeat(64),
    createdAt: "2026-08-17T12:01:00Z",
    provenance: {
      contentSha256: "b".repeat(64),
      capturedAt: "2026-08-17T12:00:30Z",
      mediaType: "text/html",
      byteSize: 2048,
      finalUrl: "https://example.org/law",
      extractedContentSha256: "a".repeat(64),
    },
    units: [
      {
        id: "unit-document",
        parentId: null,
        kind: "document",
        ordinal: 0,
        marker: null,
        label: null,
        startOffset: 0,
        endOffset: text.length,
        startPage: null,
        endPage: null,
        locator: "document",
        contentSha256: "c".repeat(64),
      },
      {
        id: "unit-title",
        parentId: "unit-document",
        kind: "title",
        ordinal: 0,
        marker: null,
        label: null,
        startOffset: 0,
        endOffset: 23,
        startPage: null,
        endPage: null,
        locator: "title",
        contentSha256: "d".repeat(64),
      },
      {
        id: "unit-article",
        parentId: "unit-document",
        kind: "article",
        ordinal: 1,
        marker: "Article 1",
        label: "Article 1",
        startOffset: 23,
        endOffset: text.length,
        startPage: null,
        endPage: null,
        locator: "article-1",
        contentSha256: "e".repeat(64),
      },
    ],
  };
}

function hierarchicalDocument(): DocumentResponse {
  const text =
    "Chapter I General provisions\nArt. 1\u00ba Article caput\nI - first item";
  const articleStart = text.indexOf("Art. 1\u00ba");
  const itemStart = text.indexOf("I - first item");
  const base = document();
  const rootUnit = base.units[0];
  const structuralUnit = base.units[1];
  if (!rootUnit || !structuralUnit) {
    throw new Error("The legal reader fixture is incomplete.");
  }
  return {
    ...base,
    text,
    units: [
      {
        ...rootUnit,
        id: "hierarchy-root",
        endOffset: text.length,
      },
      {
        ...structuralUnit,
        id: "chapter-one",
        parentId: "hierarchy-root",
        kind: "chapter",
        marker: "Chapter I General provisions",
        label: "Chapter I General provisions",
        startOffset: 0,
        endOffset: text.length,
      },
      {
        ...structuralUnit,
        id: "article-one",
        parentId: "chapter-one",
        kind: "article",
        marker: "Art. 1\u00ba",
        label: "Art. 1\u00ba",
        startOffset: articleStart,
        endOffset: text.length,
      },
      {
        ...structuralUnit,
        id: "item-one",
        parentId: "article-one",
        kind: "item",
        marker: "I -",
        label: "I -",
        startOffset: itemStart,
        endOffset: text.length,
      },
    ],
  };
}

function brazilianDocument(): DocumentResponse {
  const articleThree =
    "Art. 3\u00ba Esta Lei aplica-se desde que:\nI - a opera\u00e7\u00e3o seja realizada no territ\u00f3rio nacional;\n\u00a7 1\u00ba Consideram-se coletados no territ\u00f3rio nacional os dados do titular.\n";
  const articleFour =
    "Art. 4\u00ba Esta Lei n\u00e3o se aplica a fins particulares.";
  const text = `${articleThree}${articleFour}`;
  const articleFourStart = articleThree.length;
  const itemStart = text.indexOf("I -");
  const paragraphStart = text.indexOf("\u00a7 1\u00ba");
  const base = document();
  const rootUnit = base.units[0];
  const structuralUnit = base.units[1];
  if (!rootUnit || !structuralUnit) {
    throw new Error("The legal reader fixture is incomplete.");
  }
  return {
    ...base,
    text,
    units: [
      {
        ...rootUnit,
        id: "document-root",
        endOffset: text.length,
      },
      {
        ...structuralUnit,
        id: "article-3",
        parentId: "document-root",
        kind: "article",
        marker: "Art. 3\u00ba",
        label: "Art. 3\u00ba",
        startOffset: 0,
        endOffset: articleFourStart,
      },
      {
        ...structuralUnit,
        id: "item-1",
        parentId: "article-3",
        kind: "item",
        marker: "I -",
        label: "I -",
        startOffset: itemStart,
        endOffset: paragraphStart,
      },
      {
        ...structuralUnit,
        id: "paragraph-1",
        parentId: "article-3",
        kind: "paragraph",
        marker: "\u00a7 1\u00ba",
        label: "\u00a7 1\u00ba",
        startOffset: paragraphStart,
        endOffset: articleFourStart,
      },
      {
        ...structuralUnit,
        id: "article-4",
        parentId: "document-root",
        kind: "article",
        marker: "Art. 4\u00ba",
        label: "Art. 4\u00ba",
        startOffset: articleFourStart,
        endOffset: text.length,
      },
    ],
  };
}
