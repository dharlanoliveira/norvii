import {
  AssistantRuntimeProvider,
  AuiIf,
  ComposerPrimitive,
  ErrorPrimitive,
  MessagePrimitive,
  ThreadPrimitive,
  useLocalRuntime,
  type ChatModelAdapter,
  type SourceMessagePartProps,
  type TextMessagePartProps,
} from "@assistant-ui/react";
import { ArrowUp, BookMarked, Sparkles } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";

import type {
  CitationFixture,
  CorpusFixture,
  PreparedAnswer,
} from "../../fixtures/models";
import { resolveOpeningSuggestions } from "../../fixtures/legal-content/opening-suggestions";

interface ResearchChatProps {
  readonly corpus: CorpusFixture;
  readonly onOpenCitation: (citation: CitationFixture) => void;
}

type CitationPartProps = SourceMessagePartProps & {
  readonly onOpenCitation: (citation: CitationFixture) => void;
};

function extractPrompt(
  messages: Parameters<ChatModelAdapter["run"]>[0]["messages"],
): string {
  const userMessage = [...messages]
    .reverse()
    .find((message) => message.role === "user");
  return (
    userMessage?.content
      .filter((part) => part.type === "text")
      .map((part) => part.text)
      .join(" ") ?? ""
  );
}

function matchAnswer(
  corpus: CorpusFixture,
  prompt: string,
): PreparedAnswer | undefined {
  const normalizedPrompt = prompt.trim().toLocaleLowerCase();
  return corpus.preparedAnswers.find((candidate) =>
    candidate.prompts.some((preparedPrompt) =>
      normalizedPrompt.includes(preparedPrompt.toLocaleLowerCase()),
    ),
  );
}

function matchesFailurePrompt(corpus: CorpusFixture, prompt: string): boolean {
  const normalizedPrompt = prompt.trim().toLocaleLowerCase();
  return corpus.failurePrompts.some((failurePrompt) =>
    normalizedPrompt.includes(failurePrompt.toLocaleLowerCase()),
  );
}

function createModelAdapter(
  corpus: CorpusFixture,
  abstention: string,
): ChatModelAdapter {
  return {
    async *run({ messages, abortSignal }) {
      const prompt = extractPrompt(messages);
      const answer = matchAnswer(corpus, prompt);
      const answerText = answer?.answer ?? abstention;
      const midpoint = Math.max(1, Math.floor(answerText.length * 0.52));

      await new Promise((resolve) => window.setTimeout(resolve, 220));
      if (abortSignal.aborted) return;
      if (matchesFailurePrompt(corpus, prompt)) {
        throw new Error("Deterministic prototype response failure.");
      }
      yield {
        content: [{ type: "text", text: answerText.slice(0, midpoint) }],
      };

      await new Promise((resolve) => window.setTimeout(resolve, 260));
      yield {
        content: [
          { type: "text", text: answerText },
          ...(answer?.citations.map((citation) => ({
            type: "source" as const,
            sourceType: "document" as const,
            id: `${citation.sourceId}::${citation.sectionId}`,
            title: citation.label,
            mediaType: "text/html",
          })) ?? []),
        ],
      };
    },
  };
}

function TextPart({ text }: TextMessagePartProps) {
  return <p className="message-text">{text}</p>;
}

function CitationPart({ id, title, onOpenCitation }: CitationPartProps) {
  const { t } = useTranslation();
  const [sourceId, sectionId] = id.split("::");

  if (!sourceId || !sectionId) {
    return null;
  }

  const label = title ?? sourceId;
  return (
    <button
      type="button"
      className="citation-chip"
      aria-label={t("chat.citationLabel", { label })}
      onClick={() => onOpenCitation({ id, sourceId, sectionId, label })}
    >
      <BookMarked aria-hidden="true" size={14} />
      <span>{label}</span>
    </button>
  );
}

function UserMessage() {
  const { t } = useTranslation();
  return (
    <MessagePrimitive.Root className="chat-message user-message">
      <span className="message-author">{t("chat.userLabel")}</span>
      <MessagePrimitive.Parts components={{ Text: TextPart }} />
    </MessagePrimitive.Root>
  );
}

function AssistantMessage({
  onOpenCitation,
}: Pick<ResearchChatProps, "onOpenCitation">) {
  const { t } = useTranslation();
  return (
    <MessagePrimitive.Root className="chat-message assistant-message">
      <div className="assistant-heading">
        <span className="assistant-avatar" aria-hidden="true">
          N
        </span>
        <span>
          <strong>{t("chat.assistantLabel")}</strong>
          <small>{t("chat.simulated")}</small>
        </span>
      </div>
      <MessagePrimitive.Parts
        components={{
          Text: TextPart,
          Source: (props) => (
            <CitationPart {...props} onOpenCitation={onOpenCitation} />
          ),
        }}
      />
      <MessagePrimitive.Error>
        <ErrorPrimitive.Root className="chat-error">
          <strong>{t("chat.failureTitle")}</strong>
          <span>{t("chat.failureBody")}</span>
        </ErrorPrimitive.Root>
      </MessagePrimitive.Error>
    </MessagePrimitive.Root>
  );
}

function ChatThread({ corpus, onOpenCitation }: ResearchChatProps) {
  const { i18n, t } = useTranslation();
  const interfaceLanguage = i18n.resolvedLanguage === "pt" ? "pt" : "en";
  const openingSuggestions = resolveOpeningSuggestions(
    corpus,
    interfaceLanguage,
  );

  return (
    <ThreadPrimitive.Root
      className="research-chat"
      aria-label={t("chat.regionLabel")}
    >
      <ThreadPrimitive.Viewport className="chat-viewport" aria-live="polite">
        <AuiIf condition={(state) => state.thread.isEmpty}>
          <div className="chat-empty reveal">
            <span className="chat-empty-icon" aria-hidden="true">
              <Sparkles size={20} />
            </span>
            <p className="eyebrow">{t("chat.kicker")}</p>
            <h2>{t("chat.title")}</h2>
            <p>{t("chat.emptyBody")}</p>
            {openingSuggestions.length > 0 ? (
              <div className="suggestion-list">
                {openingSuggestions.map((suggestion) => (
                  <ThreadPrimitive.Suggestion
                    className="suggestion-button"
                    prompt={suggestion.question}
                    send
                    key={suggestion.caseId}
                  >
                    <span>{String(suggestion.rank).padStart(2, "0")}</span>
                    {suggestion.question}
                  </ThreadPrimitive.Suggestion>
                ))}
              </div>
            ) : null}
          </div>
        </AuiIf>
        <ThreadPrimitive.Messages>
          {({ message }) =>
            message.role === "user" ? (
              <UserMessage />
            ) : (
              <AssistantMessage onOpenCitation={onOpenCitation} />
            )
          }
        </ThreadPrimitive.Messages>
        <AuiIf condition={(state) => state.thread.isRunning}>
          <div className="chat-responding" role="status">
            <span aria-hidden="true" />
            {t("chat.responding")}
          </div>
        </AuiIf>
        <ThreadPrimitive.ViewportFooter className="composer-dock">
          <ComposerPrimitive.Root className="chat-composer">
            <label className="visually-hidden" htmlFor="research-question">
              {t("chat.composerLabel")}
            </label>
            <ComposerPrimitive.Input
              id="research-question"
              rows={1}
              placeholder={t("chat.composerPlaceholder")}
              aria-label={t("chat.composerLabel")}
              className="composer-input"
            />
            <ComposerPrimitive.Send
              className="composer-send"
              aria-label={t("chat.send")}
            >
              <ArrowUp aria-hidden="true" size={18} strokeWidth={2} />
            </ComposerPrimitive.Send>
          </ComposerPrimitive.Root>
          <div className="composer-meta">
            <span>{corpus.label}</span>
            <span>{t("status.localData")}</span>
          </div>
        </ThreadPrimitive.ViewportFooter>
      </ThreadPrimitive.Viewport>
    </ThreadPrimitive.Root>
  );
}

export function ResearchChat({ corpus, onOpenCitation }: ResearchChatProps) {
  const { t } = useTranslation();
  const adapter = useMemo(
    () => createModelAdapter(corpus, t("chat.abstention")),
    [corpus, t],
  );
  const runtime = useLocalRuntime(adapter);

  return (
    <AssistantRuntimeProvider runtime={runtime}>
      <ChatThread corpus={corpus} onOpenCitation={onOpenCitation} />
    </AssistantRuntimeProvider>
  );
}
