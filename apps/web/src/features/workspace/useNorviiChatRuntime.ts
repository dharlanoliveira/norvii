import {
  type AssistantRuntime,
  type ChatModelAdapter,
  useLocalRuntime,
} from "@assistant-ui/react";
import { useCallback, useMemo, useState } from "react";

import type {
  ChatInspection,
  ChatProvider,
  ChatReference,
  ChatStreamEvent,
  RetrievalStrategy,
} from "../../api/chat";
import type { CorpusLanguage } from "../../api/contract";

interface UseNorviiChatRuntimeOptions {
  readonly corpusId: string;
  readonly provider: ChatProvider;
  readonly interfaceLanguage: CorpusLanguage;
  readonly abstainedAnswer: string;
  readonly fallbackError: string;
  readonly strategy: RetrievalStrategy;
}

interface NorviiChatRuntime {
  readonly runtime: AssistantRuntime;
  readonly terminalStatesByMessageId: ReadonlyMap<
    string,
    AssistantTerminalState
  >;
  readonly referencesByMessageId: ReadonlyMap<string, readonly ChatReference[]>;
  readonly inspectionsByMessageId: ReadonlyMap<string, ChatInspection>;
  readonly lastSubmittedQuestion: string | undefined;
  readonly lastSubmittedQuestionVersion: number;
}

export type AssistantTerminalState =
  | {
      readonly kind: "cancelled";
    }
  | {
      readonly kind: "error";
      readonly message: string;
    };

export function useNorviiChatRuntime({
  corpusId,
  provider,
  interfaceLanguage,
  abstainedAnswer,
  fallbackError,
  strategy,
}: UseNorviiChatRuntimeOptions): NorviiChatRuntime {
  const [terminalStatesByMessageId, setTerminalStatesByMessageId] = useState<
    ReadonlyMap<string, AssistantTerminalState>
  >(() => new Map());
  const [referencesByMessageId, setReferencesByMessageId] = useState<
    ReadonlyMap<string, readonly ChatReference[]>
  >(() => new Map());
  const [inspectionsByMessageId, setInspectionsByMessageId] = useState<
    ReadonlyMap<string, ChatInspection>
  >(() => new Map());
  const [lastSubmittedQuestion, setLastSubmittedQuestion] = useState<string>();
  const [lastSubmittedQuestionVersion, setLastSubmittedQuestionVersion] =
    useState(0);

  const storeTerminalState = useCallback(
    (messageId: string | undefined, terminalState: AssistantTerminalState) => {
      if (messageId === undefined) return;
      setTerminalStatesByMessageId((current) => {
        const next = new Map(current);
        next.set(messageId, terminalState);
        return next;
      });
    },
    [],
  );

  const storeReferences = useCallback(
    (messageId: string | undefined, references: readonly ChatReference[]) => {
      if (messageId === undefined) return;
      setReferencesByMessageId((current) => {
        const next = new Map(current);
        next.set(messageId, references);
        return next;
      });
    },
    [],
  );

  const storeInspection = useCallback(
    (messageId: string | undefined, inspection: ChatInspection | undefined) => {
      if (messageId === undefined || inspection?.outcome !== "completed")
        return;
      setInspectionsByMessageId((current) => {
        const next = new Map(current);
        next.set(messageId, inspection);
        return next;
      });
    },
    [],
  );

  const adapter = useMemo<ChatModelAdapter>(
    () => ({
      async *run(options) {
        const question = latestQuestion(options.messages);
        if (question === "") {
          storeTerminalState(options.unstable_assistantMessageId, {
            kind: "error",
            message: fallbackError,
          });
          yield textResult(fallbackError);
          return;
        }
        setLastSubmittedQuestion(question);
        setLastSubmittedQuestionVersion((current) => current + 1);

        const assistantMessageId = options.unstable_assistantMessageId;
        const stream = new ChatStreamQueue();
        void provider
          .streamQuestion(
            corpusId,
            question,
            interfaceLanguage,
            strategy,
            options.abortSignal,
            (event) => stream.push(event),
          )
          .then(() => stream.close())
          .catch((reason: unknown) =>
            stream.fail(
              reason instanceof Error ? reason : new Error(fallbackError),
            ),
          );

        let answer = "";
        try {
          for await (const event of stream) {
            switch (event.type) {
              case "started":
                break;
              case "evidence":
                storeReferences(assistantMessageId, event.references);
                break;
              case "delta":
                answer += event.text;
                yield textResult(answer);
                break;
              case "completed":
                storeReferences(assistantMessageId, event.references);
                storeInspection(assistantMessageId, event.inspection);
                yield textResult(event.answer);
                return;
              case "abstained":
                yield textResult(abstainedAnswer);
                return;
              case "cancelled":
                storeTerminalState(assistantMessageId, { kind: "cancelled" });
                yield textResult("");
                return;
              case "error": {
                const message = event.message;
                storeTerminalState(assistantMessageId, {
                  kind: "error",
                  message,
                });
                yield textResult(message);
                return;
              }
            }
          }
        } catch (reason: unknown) {
          if (options.abortSignal.aborted) return;
          const message =
            reason instanceof Error ? reason.message : fallbackError;
          storeTerminalState(assistantMessageId, { kind: "error", message });
          yield textResult(message);
          return;
        }

        if (options.abortSignal.aborted) return;
        storeTerminalState(assistantMessageId, {
          kind: "error",
          message: fallbackError,
        });
        yield textResult(fallbackError);
      },
    }),
    [
      abstainedAnswer,
      corpusId,
      fallbackError,
      interfaceLanguage,
      provider,
      storeReferences,
      storeInspection,
      storeTerminalState,
      strategy,
    ],
  );
  const runtime = useLocalRuntime(adapter);

  return {
    runtime,
    terminalStatesByMessageId,
    referencesByMessageId,
    inspectionsByMessageId,
    lastSubmittedQuestion,
    lastSubmittedQuestionVersion,
  };
}

function latestQuestion(
  messages: Parameters<ChatModelAdapter["run"]>[0]["messages"],
): string {
  const latestUserMessage = [...messages]
    .reverse()
    .find((message) => message.role === "user");
  if (latestUserMessage === undefined) return "";

  return latestUserMessage.content
    .filter((part) => part.type === "text")
    .map((part) => part.text)
    .join("\n")
    .trim();
}

function textResult(text: string) {
  return { content: [{ type: "text" as const, text }] };
}

class ChatStreamQueue implements AsyncIterable<ChatStreamEvent> {
  readonly #events: ChatStreamEvent[] = [];
  #resolve: ((result: IteratorResult<ChatStreamEvent>) => void) | undefined;
  #reject: ((reason?: unknown) => void) | undefined;
  #closed = false;
  #failure: Error | undefined;

  push(event: ChatStreamEvent): void {
    if (this.#closed) return;
    if (this.#resolve !== undefined) {
      const resolve = this.#resolve;
      this.#resolve = undefined;
      this.#reject = undefined;
      resolve({ value: event, done: false });
      return;
    }
    this.#events.push(event);
  }

  close(): void {
    if (this.#closed) return;
    this.#closed = true;
    this.#resolve?.({ value: undefined, done: true });
    this.#resolve = undefined;
    this.#reject = undefined;
  }

  fail(reason: Error): void {
    if (this.#closed) return;
    this.#closed = true;
    this.#failure = reason;
    this.#reject?.(reason);
    this.#resolve = undefined;
    this.#reject = undefined;
  }

  [Symbol.asyncIterator](): AsyncIterator<ChatStreamEvent> {
    return { next: () => this.next() };
  }

  private next(): Promise<IteratorResult<ChatStreamEvent>> {
    const event = this.#events.shift();
    if (event !== undefined)
      return Promise.resolve({ value: event, done: false });
    if (this.#failure !== undefined) return Promise.reject(this.#failure);
    if (this.#closed) return Promise.resolve({ value: undefined, done: true });

    return new Promise<IteratorResult<ChatStreamEvent>>((resolve, reject) => {
      this.#resolve = resolve;
      this.#reject = reject;
    });
  }
}
