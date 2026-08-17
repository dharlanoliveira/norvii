import type { Corpus, ResponsePart } from "../domain/models";

export type PreparedResponseResult =
  | {
      readonly status: "answered";
      readonly parts: readonly ResponsePart[];
    }
  | {
      readonly status: "abstained";
      readonly parts: readonly [];
    }
  | {
      readonly status: "failed";
      readonly code: "prepared-response-failed";
      readonly parts: readonly [];
    };

export interface PreparedResponseEngine {
  resolve(corpus: Corpus, question: string): PreparedResponseResult;
}

export function createPreparedResponseEngine(): PreparedResponseEngine {
  return {
    resolve: (corpus, question) => {
      const normalizedQuestion = question
        .trim()
        .toLocaleLowerCase(corpus.language);
      const response = corpus.preparedResponses.find((candidate) =>
        candidate.prompts.some((prompt) =>
          normalizedQuestion.includes(
            prompt.toLocaleLowerCase(corpus.language),
          ),
        ),
      );

      if (!response) return { status: "abstained", parts: [] };
      if (response.outcome === "failed") {
        return {
          status: "failed",
          code: "prepared-response-failed",
          parts: [],
        };
      }
      return { status: "answered", parts: response.parts };
    },
  };
}
