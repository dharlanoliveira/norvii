import {
  AssistantRuntimeProvider,
  ActionBarPrimitive,
  AuiIf,
  ComposerPrimitive,
  ErrorPrimitive,
  MessagePrimitive,
  ThreadPrimitive,
  useLocalRuntime,
  type AssistantState,
  type SourceMessagePartProps,
  type TextMessagePartProps,
} from "@assistant-ui/react";
import { ArrowUp, BookMarked, RotateCcw, Sparkles } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";

import type { Citation, Corpus } from "../../research/domain/models";
import { createAssistantAdapter } from "../../research/adapters/createAssistantAdapter";
import { createPreparedResponseEngine } from "../../research/demonstration/preparedResponseEngine";

interface ResearchChatProps {
  readonly corpus: Corpus;
  readonly onOpenCitation: (citation: Citation) => void;
}

const selectEmptyThread = (state: AssistantState) => state.thread.isEmpty;
const selectRunningThread = (state: AssistantState) => state.thread.isRunning;

function TextPart({ text }: TextMessagePartProps) {
  return <p className="message-text">{text}</p>;
}

type CitationPartProps = SourceMessagePartProps & {
  readonly onOpenCitation: (citation: Citation) => void;
};

function CitationPart({ id, title, onOpenCitation }: CitationPartProps) {
  const { t } = useTranslation();
  const [citationId, sourceId, locationId] = id.split("::");

  if (!citationId || !sourceId || !locationId) return null;

  const label = title ?? sourceId;
  return (
    <button
      type="button"
      className="citation-chip"
      aria-label={t("chat.citationLabel", { label })}
      onClick={() =>
        onOpenCitation({
          id: citationId,
          sourceId,
          locationId,
          label,
        })
      }
    >
      <BookMarked aria-hidden="true" size={14} />
      <span>{label}</span>
    </button>
  );
}

function UserMessage() {
  const { t } = useTranslation();
  return (
    <MessagePrimitive.Root className="chat-message chat-message--user">
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
    <MessagePrimitive.Root className="chat-message chat-message--assistant">
      <div className="assistant-identity">
        <span aria-hidden="true">N</span>
        <div>
          <strong>{t("chat.assistantLabel")}</strong>
          <small>{t("chat.simulated")}</small>
        </div>
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
        <ErrorPrimitive.Root className="chat-error" role="alert">
          <strong>{t("chat.failureTitle")}</strong>
          <span>{t("chat.failureBody")}</span>
          <ActionBarPrimitive.Root>
            <ActionBarPrimitive.Reload className="chat-error__retry">
              <RotateCcw aria-hidden="true" size={13} />
              {t("chat.retry")}
            </ActionBarPrimitive.Reload>
          </ActionBarPrimitive.Root>
        </ErrorPrimitive.Root>
      </MessagePrimitive.Error>
    </MessagePrimitive.Root>
  );
}

function ChatThread({ corpus, onOpenCitation }: ResearchChatProps) {
  const { t } = useTranslation();

  return (
    <ThreadPrimitive.Root
      className="research-chat"
      aria-label={t("chat.regionLabel")}
    >
      <ThreadPrimitive.Viewport className="chat-viewport" aria-live="polite">
        <AuiIf condition={selectEmptyThread}>
          <div className="chat-empty reveal">
            <span className="chat-empty__icon" aria-hidden="true">
              <Sparkles size={20} />
            </span>
            <p className="kicker">{t("chat.kicker")}</p>
            <h2>{t("chat.title")}</h2>
            <p>{t("chat.emptyBody")}</p>
            <div className="suggestion-list">
              {corpus.suggestedQuestions.map((question, index) => (
                <ThreadPrimitive.Suggestion
                  className="suggestion-button"
                  prompt={question}
                  send
                  key={question}
                >
                  <span>{String(index + 1).padStart(2, "0")}</span>
                  {question}
                </ThreadPrimitive.Suggestion>
              ))}
            </div>
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
        <AuiIf condition={selectRunningThread}>
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
              <ArrowUp aria-hidden="true" size={18} />
            </ComposerPrimitive.Send>
          </ComposerPrimitive.Root>
          <div className="composer-meta">
            <span>{corpus.name}</span>
            <span>{t("chat.localData")}</span>
          </div>
        </ThreadPrimitive.ViewportFooter>
      </ThreadPrimitive.Viewport>
    </ThreadPrimitive.Root>
  );
}

export function ResearchChat({ corpus, onOpenCitation }: ResearchChatProps) {
  const { t } = useTranslation();
  const engine = useMemo(() => createPreparedResponseEngine(), []);
  const adapter = useMemo(
    () =>
      createAssistantAdapter(
        corpus,
        engine,
        t("chat.abstention"),
        t("chat.retryComplete"),
      ),
    [corpus, engine, t],
  );
  const runtime = useLocalRuntime(adapter);

  return (
    <AssistantRuntimeProvider runtime={runtime}>
      <ChatThread corpus={corpus} onOpenCitation={onOpenCitation} />
    </AssistantRuntimeProvider>
  );
}
