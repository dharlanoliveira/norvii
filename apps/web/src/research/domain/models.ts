export type InterfaceLanguage = "en" | "pt";

export interface SourceLocation {
  readonly id: string;
  readonly label: string;
  readonly content: string;
  readonly page?: number;
}

interface BaseSource {
  readonly id: string;
  readonly corpusId: string;
  readonly title: string;
  readonly authority: string;
  readonly officialReference: string;
  readonly locations: readonly SourceLocation[];
}

export interface PdfSource extends BaseSource {
  readonly kind: "pdf";
  readonly pageCount: number;
}

export type ExternalPreview =
  | { readonly status: "available"; readonly summary: string }
  | { readonly status: "unavailable"; readonly reason: string };

export interface ExternalSource extends BaseSource {
  readonly kind: "external";
  readonly url: string;
  readonly preview: ExternalPreview;
}

export type Source = PdfSource | ExternalSource;

export interface Citation {
  readonly id: string;
  readonly sourceId: string;
  readonly locationId: string;
  readonly label: string;
}

export type ResponsePart =
  | { readonly type: "text"; readonly text: string }
  | { readonly type: "citation"; readonly citation: Citation };

export interface PreparedResponse {
  readonly id: string;
  readonly prompts: readonly string[];
  readonly outcome: "answered" | "failed";
  readonly parts: readonly ResponsePart[];
}

export interface Corpus {
  readonly id: string;
  readonly language: InterfaceLanguage;
  readonly name: string;
  readonly jurisdiction: string;
  readonly summary: string;
  readonly sources: readonly Source[];
  readonly suggestedQuestions: readonly string[];
  readonly preparedResponses: readonly PreparedResponse[];
}

export interface CorpusSummary {
  readonly id: string;
  readonly language: InterfaceLanguage;
  readonly name: string;
  readonly jurisdiction: string;
  readonly summary: string;
  readonly sourceCount: number;
}

export interface ResolvedCitation {
  readonly source: Source;
  readonly location: SourceLocation;
}
