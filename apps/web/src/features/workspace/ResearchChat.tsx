import {
  AssistantRuntimeProvider,
  AuiIf,
  ComposerPrimitive,
  MessagePartPrimitive,
  MessagePrimitive,
  ThreadPrimitive,
  useAuiState,
} from "@assistant-ui/react";
import { MessageSquare, Send, Square } from "lucide-react";
import { useTranslation } from "react-i18next";

import {
  createHttpChatProvider,
  type ChatProvider,
  type ChatReference,
} from "../../api/chat";
import { AssistantMarkdown } from "./AssistantMarkdown";
import { useNorviiChatRuntime } from "./useNorviiChatRuntime";

const defaultChatProvider = createHttpChatProvider();

interface ResearchChatProps {
  readonly corpusId: string;
  readonly provider?: ChatProvider | undefined;
  readonly onReferenceSelect?: ((reference: ChatReference) => void) | undefined;
}

export function ResearchChat({
  corpusId,
  provider = defaultChatProvider,
  onReferenceSelect,
}: ResearchChatProps) {
  const { t, i18n } = useTranslation();
  const interfaceLanguage: "en" | "pt" = i18n.resolvedLanguage?.startsWith("pt")
    ? "pt"
    : "en";
  const { runtime, error, referencesByMessageId } = useNorviiChatRuntime({
    corpusId,
    provider,
    interfaceLanguage,
    abstainedAnswer: t("chat.abstained"),
    fallbackError: t("chat.failed"),
  });

  return (
    <AssistantRuntimeProvider runtime={runtime}>
      <section className="research-chat" aria-label={t("chat.regionLabel")}>
        <ThreadPrimitive.Root className="chat-thread">
          <ThreadPrimitive.Viewport className="chat-viewport">
            <AuiIf condition={(state) => state.thread.isEmpty}>
              <ChatEmptyState />
            </AuiIf>
            <ThreadPrimitive.Messages>
              {({ message }) =>
                message.role === "user" ? (
                  <UserMessage />
                ) : (
                  <AssistantMessage
                    onReferenceSelect={onReferenceSelect}
                    referencesByMessageId={referencesByMessageId}
                  />
                )
              }
            </ThreadPrimitive.Messages>
            {error ? <ChatError message={error} /> : null}
          </ThreadPrimitive.Viewport>
          <ChatComposer interfaceLanguage={interfaceLanguage} />
        </ThreadPrimitive.Root>
      </section>
    </AssistantRuntimeProvider>
  );
}

function ChatEmptyState() {
  const { t } = useTranslation();
  return (
    <div className="chat-empty">
      <div className="chat-empty__icon">
        <MessageSquare aria-hidden="true" size={24} />
      </div>
      <p className="kicker">{t("chat.kicker")}</p>
      <h2>{t("chat.title")}</h2>
      <p>{t("chat.introduction")}</p>
    </div>
  );
}

function UserMessage() {
  const { t } = useTranslation();
  return (
    <MessagePrimitive.Root className="chat-message chat-message--user">
      <span className="message-author">{t("chat.you")}</span>
      <MessagePrimitive.Parts components={{ Text: UserMessageText }} />
    </MessagePrimitive.Root>
  );
}

function UserMessageText() {
  return (
    <p className="message-text">
      <MessagePartPrimitive.Text />
    </p>
  );
}

interface AssistantMessageProps {
  readonly referencesByMessageId: ReadonlyMap<string, readonly ChatReference[]>;
  readonly onReferenceSelect?: ((reference: ChatReference) => void) | undefined;
}

function AssistantMessage({
  referencesByMessageId,
  onReferenceSelect,
}: AssistantMessageProps) {
  const { t } = useTranslation();
  const messageId = useAuiState((state) => state.message.id);
  const references = referencesByMessageId.get(messageId) ?? [];
  return (
    <MessagePrimitive.Root className="chat-message chat-message--assistant">
      <span className="message-author">{t("chat.assistant")}</span>
      <MessagePrimitive.Parts
        components={{ Text: AssistantMarkdown, Empty: AssistantPending }}
      />
      {references.length > 0 ? (
        <div className="citation-list" aria-label={t("chat.references")}>
          {references.map((reference) => (
            <button
              className="citation-chip"
              key={reference.id}
              type="button"
              onClick={() => onReferenceSelect?.(reference)}
            >
              [{reference.rank}] {reference.unitLocator}
            </button>
          ))}
        </div>
      ) : null}
    </MessagePrimitive.Root>
  );
}

function AssistantPending() {
  const { t } = useTranslation();
  return <p className="message-text">{t("chat.responding")}</p>;
}

function ChatError({ message }: { readonly message: string }) {
  const { t } = useTranslation();
  return (
    <div className="chat-error" role="alert">
      <strong>{t("chat.errorTitle")}</strong>
      <span>{message}</span>
    </div>
  );
}

function ChatComposer({
  interfaceLanguage,
}: {
  readonly interfaceLanguage: "en" | "pt";
}) {
  const { t } = useTranslation();
  return (
    <div className="composer-dock">
      <ComposerPrimitive.Root className="chat-composer">
        <ComposerPrimitive.Input
          aria-label={t("chat.questionLabel")}
          className="composer-input"
          placeholder={t("chat.placeholder")}
          rows={2}
          submitMode="ctrlEnter"
        />
        <AuiIf condition={(state) => state.thread.isRunning}>
          <ComposerPrimitive.Cancel
            aria-label={t("chat.stop")}
            className="composer-send"
          >
            <Square aria-hidden="true" size={15} />
          </ComposerPrimitive.Cancel>
        </AuiIf>
        <AuiIf condition={(state) => !state.thread.isRunning}>
          <ComposerPrimitive.Send
            aria-label={t("chat.send")}
            className="composer-send"
          >
            <Send aria-hidden="true" size={15} />
          </ComposerPrimitive.Send>
        </AuiIf>
      </ComposerPrimitive.Root>
      <div className="composer-meta">
        <span>{t("chat.groundedOnly")}</span>
        <span>
          {t("chat.language", {
            language: t(`language.${interfaceLanguage}`),
          })}
        </span>
      </div>
    </div>
  );
}
