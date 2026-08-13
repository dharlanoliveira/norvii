import type { ChatModelAdapter } from "@assistant-ui/react";

import type { Corpus } from "../domain/models";
import type { PreparedResponseEngine } from "../demonstration/preparedResponseEngine";

function extractQuestion(
  messages: Parameters<ChatModelAdapter["run"]>[0]["messages"],
): string {
  const message = [...messages]
    .reverse()
    .find((candidate) => candidate.role === "user");

  return (
    message?.content
      .filter((part) => part.type === "text")
      .map((part) => part.text)
      .join(" ") ?? ""
  );
}

export function createAssistantAdapter(
  corpus: Corpus,
  engine: PreparedResponseEngine,
  abstention: string,
  retryComplete: string,
): ChatModelAdapter {
  const failedQuestions = new Set<string>();

  return {
    async *run({ messages, abortSignal }) {
      const question = extractQuestion(messages);
      const result = engine.resolve(corpus, question);

      await new Promise((resolve) => window.setTimeout(resolve, 35));
      if (abortSignal.aborted) return;
      if (result.status === "failed") {
        if (!failedQuestions.has(question)) {
          failedQuestions.add(question);
          throw new Error(result.code);
        }
        yield {
          content: [{ type: "text" as const, text: retryComplete }],
        };
        return;
      }

      const parts =
        result.status === "abstained"
          ? [{ type: "text" as const, text: abstention }]
          : result.parts.map((part) =>
              part.type === "text"
                ? { type: "text" as const, text: part.text }
                : {
                    type: "source" as const,
                    sourceType: "document" as const,
                    id: [
                      part.citation.id,
                      part.citation.sourceId,
                      part.citation.locationId,
                    ].join("::"),
                    title: part.citation.label,
                    mediaType: "text/plain",
                  },
            );

      yield { content: parts };
    },
  };
}
