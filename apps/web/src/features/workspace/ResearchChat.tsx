import {
  AssistantRuntimeProvider,
  AuiIf,
  ComposerPrimitive,
  MessagePartPrimitive,
  MessagePrimitive,
  ThreadPrimitive,
  useAui,
  useAuiState,
} from "@assistant-ui/react";
import { ChevronDown, Info, Send, Square } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import {
  createHttpChatProvider,
  type ChatProvider,
  type ChatInspection,
  type ChatReference,
  type RetrievalStrategy,
} from "../../api/chat";
import { AssistantMarkdown } from "./AssistantMarkdown";
import { StrategyComparison } from "./StrategyComparison";
import { useNorviiChatRuntime } from "./useNorviiChatRuntime";

const defaultChatProvider = createHttpChatProvider();
const initiallyVisibleCitationLocations = 3;

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
  const [strategy, setStrategy] = useState<RetrievalStrategy>("vector");
  const {
    runtime,
    error,
    referencesByMessageId,
    inspectionsByMessageId,
    lastSubmittedQuestion,
  } = useNorviiChatRuntime({
    corpusId,
    provider,
    interfaceLanguage,
    abstainedAnswer: t("chat.abstained"),
    fallbackError: t("chat.failed"),
    strategy,
  });

  return (
    <AssistantRuntimeProvider runtime={runtime}>
      <section className="research-chat" aria-label={t("chat.regionLabel")}>
        <label className="chat-strategy">
          <span>{t("chat.strategy.label")}</span>
          <select
            aria-label={t("chat.strategy.label")}
            value={strategy}
            onChange={(event) =>
              setStrategy(event.target.value as RetrievalStrategy)
            }
          >
            <option value="vector">{t("chat.strategy.vector")}</option>
            <option value="graph">{t("chat.strategy.graph")}</option>
            <option value="hybrid">{t("chat.strategy.hybrid")}</option>
          </select>
        </label>
        <ThreadPrimitive.Root className="chat-thread">
          <ThreadPrimitive.Viewport className="chat-viewport" turnAnchor="top">
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
                    inspectionsByMessageId={inspectionsByMessageId}
                  />
                )
              }
            </ThreadPrimitive.Messages>
            {error ? <ChatError message={error} /> : null}
            <ChatViewportFooter
              corpusId={corpusId}
              interfaceLanguage={interfaceLanguage}
              lastSubmittedQuestion={lastSubmittedQuestion}
              onReferenceSelect={onReferenceSelect}
              provider={provider}
            />
          </ThreadPrimitive.Viewport>
        </ThreadPrimitive.Root>
      </section>
    </AssistantRuntimeProvider>
  );
}

function ChatEmptyState() {
  const { t } = useTranslation();
  return (
    <div className="chat-empty">
      <h2>{t("chat.title")}</h2>
      <p>{t("chat.introduction")}</p>
    </div>
  );
}

function ChatViewportFooter({
  corpusId,
  interfaceLanguage,
  lastSubmittedQuestion,
  onReferenceSelect,
  provider,
}: {
  readonly corpusId: string;
  readonly interfaceLanguage: "en" | "pt";
  readonly lastSubmittedQuestion: string | undefined;
  readonly onReferenceSelect?: ((reference: ChatReference) => void) | undefined;
  readonly provider: ChatProvider;
}) {
  const isEmpty = useAuiState((state) => state.thread.isEmpty);
  return (
    <ThreadPrimitive.ViewportFooter
      className={`chat-viewport__footer${isEmpty ? " chat-viewport__footer--empty" : ""}`}
    >
      {!isEmpty ? (
        <StrategyComparison
          corpusId={corpusId}
          interfaceLanguage={interfaceLanguage}
          onReferenceSelect={onReferenceSelect}
          provider={provider}
          question={lastSubmittedQuestion}
        />
      ) : null}
      <ChatComposer interfaceLanguage={interfaceLanguage} />
      {isEmpty ? <ChatStarterQuestions /> : null}
    </ThreadPrimitive.ViewportFooter>
  );
}

function ChatStarterQuestions() {
  const { t } = useTranslation();
  const aui = useAui();
  const isRunning = useAuiState((state) => state.thread.isRunning);
  const questions = [
    t("chat.starterQuestions.purpose"),
    t("chat.starterQuestions.scope"),
    t("chat.starterQuestions.rights"),
  ];

  return (
    <div
      aria-label={t("chat.starterQuestionsLabel")}
      className="chat-starter-questions"
    >
      {questions.map((question) => (
        <button
          className="chat-starter-question"
          disabled={isRunning}
          key={question}
          type="button"
          onClick={() => {
            aui.thread.append({
              content: [{ type: "text", text: question }],
              role: "user",
            });
          }}
        >
          {question}
        </button>
      ))}
    </div>
  );
}

function UserMessage() {
  const { t } = useTranslation();
  return (
    <MessagePrimitive.Root
      aria-label={t("chat.you")}
      className="chat-message chat-message--user"
      role="article"
    >
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
  readonly inspectionsByMessageId: ReadonlyMap<string, ChatInspection>;
}

function AssistantMessage({
  referencesByMessageId,
  inspectionsByMessageId,
  onReferenceSelect,
}: AssistantMessageProps) {
  const { t } = useTranslation();
  const messageId = useAuiState((state) => state.message.id);
  const references = referencesByMessageId.get(messageId) ?? [];
  const inspection = inspectionsByMessageId.get(messageId);
  return (
    <MessagePrimitive.Root
      aria-label={t("chat.assistant")}
      className="chat-message chat-message--assistant"
      role="article"
    >
      <span className="message-author">{t("chat.assistant")}</span>
      <MessagePrimitive.Parts
        components={{ Text: AssistantMarkdown, Empty: AssistantPending }}
      />
      {references.length > 0 ? (
        <CitationList
          onReferenceSelect={onReferenceSelect}
          references={references}
        />
      ) : null}
      {inspection ? (
        <AnswerInspection
          inspection={inspection}
          onReferenceSelect={onReferenceSelect}
        />
      ) : null}
    </MessagePrimitive.Root>
  );
}

interface CitationLocationGroup {
  readonly id: string;
  readonly ranks: readonly number[];
  readonly reference: ChatReference;
  readonly supportingPassageCount: number;
}

function CitationList({
  references,
  onReferenceSelect,
}: {
  readonly references: readonly ChatReference[];
  readonly onReferenceSelect?: ((reference: ChatReference) => void) | undefined;
}) {
  const { t } = useTranslation();
  const [showAllLocations, setShowAllLocations] = useState(false);
  const [selectedLocationId, setSelectedLocationId] = useState<string>();
  const citationGroups = groupCitationLocations(references);
  const visibleGroups = showAllLocations
    ? citationGroups
    : citationGroups.slice(0, initiallyVisibleCitationLocations);
  const hiddenLocationCount = citationGroups.length - visibleGroups.length;

  return (
    <section className="citation-list" aria-label={t("chat.citationLocations")}>
      <div className="citation-list__items">
        {visibleGroups.map((group) => {
          const sourceTitle =
            group.reference.sourceTitle ?? group.reference.sourceId;
          const location = formatCitationLocation(
            group.reference.unitLocator,
            t,
          );
          const isSelected = group.id === selectedLocationId;
          return (
            <button
              aria-current={isSelected ? "location" : undefined}
              aria-label={t("chat.openCitation", {
                location,
                source: sourceTitle,
              })}
              className="citation-chip"
              key={group.id}
              type="button"
              onClick={() => {
                setSelectedLocationId(group.id);
                onReferenceSelect?.(group.reference);
              }}
            >
              <span className="citation-chip__identity">
                <span className="citation-chip__ranks">
                  {formatCitationRanks(group.ranks)}
                </span>
                <span>{sourceTitle}</span>
                <span aria-hidden="true" className="citation-chip__separator">
                  /
                </span>
                <strong>{location}</strong>
              </span>
              <span className="citation-chip__count">
                {t("chat.supportingPassages", {
                  count: group.supportingPassageCount,
                })}
              </span>
            </button>
          );
        })}
      </div>
      {hiddenLocationCount > 0 ? (
        <button
          className="citation-list__reveal"
          type="button"
          onClick={() => setShowAllLocations(true)}
        >
          {t("chat.showMoreLocations", { count: hiddenLocationCount })}
        </button>
      ) : null}
      {showAllLocations &&
      citationGroups.length > initiallyVisibleCitationLocations ? (
        <button
          className="citation-list__reveal"
          type="button"
          onClick={() => setShowAllLocations(false)}
        >
          {t("chat.showFewerLocations")}
        </button>
      ) : null}
    </section>
  );
}

function groupCitationLocations(
  references: readonly ChatReference[],
): readonly CitationLocationGroup[] {
  const groups = new Map<string, CitationLocationGroup>();
  for (const reference of references) {
    const groupId = [
      reference.sourceId,
      reference.documentVersionId ?? reference.documentId,
      citationLocatorKey(reference.unitLocator),
    ].join("\u001f");
    const existingGroup = groups.get(groupId);
    if (existingGroup === undefined) {
      groups.set(groupId, {
        id: groupId,
        ranks: [reference.rank],
        reference,
        supportingPassageCount: 1,
      });
      continue;
    }
    groups.set(groupId, {
      ...existingGroup,
      ranks: [...existingGroup.ranks, reference.rank],
      supportingPassageCount: existingGroup.supportingPassageCount + 1,
    });
  }
  return [...groups.values()].sort(
    (left, right) => left.reference.rank - right.reference.rank,
  );
}

function citationLocatorKey(locator: string): string {
  const articleNumber = citationArticleNumber(locator);
  return articleNumber === undefined
    ? locator.trim().replaceAll(/\s+/gu, " ").toLocaleLowerCase()
    : `article:${articleNumber.toLocaleLowerCase()}`;
}

function formatCitationRanks(ranks: readonly number[]): string {
  return `[${ranks.join(", ")}]`;
}

function formatCitationLocation(
  locator: string,
  t: ReturnType<typeof useTranslation>["t"],
): string {
  const articleNumber = citationArticleNumber(locator);
  return articleNumber === undefined
    ? locator
    : t("chat.articleLocation", { number: articleNumber });
}

function citationArticleNumber(locator: string): string | undefined {
  const trimmedLocator = locator.trim();
  const prefix = ["article", "artigo", "art."].find((candidate) =>
    trimmedLocator.toLocaleLowerCase().startsWith(candidate),
  );
  if (prefix === undefined) return undefined;

  const suffix = trimmedLocator.slice(prefix.length);
  if (suffix === "" || !isArticleSeparator(suffix.at(0))) return undefined;

  const articleNumber = removeArticleSeparators(suffix);
  return articleNumber === "" ? undefined : articleNumber;
}

function isArticleSeparator(character: string | undefined): boolean {
  return character === "-" || character?.trim() === "";
}

function removeArticleSeparators(suffix: string): string {
  let index = 0;
  while (isArticleSeparator(suffix.at(index))) index += 1;
  return suffix.slice(index);
}

function AnswerInspection({
  inspection,
  onReferenceSelect,
}: {
  readonly inspection: ChatInspection;
  readonly onReferenceSelect?: ((reference: ChatReference) => void) | undefined;
}) {
  const { t } = useTranslation();
  const evidence = inspection.evidence ?? [];
  const snapshotId = evidence.find(
    (reference) => reference.snapshotId !== undefined,
  )?.snapshotId;
  return (
    <details className="answer-inspection">
      <summary>
        <span className="answer-inspection__trigger">
          <Info aria-hidden="true" size={14} />
          {t("chat.inspect")}
        </span>
        <ChevronDown
          aria-hidden="true"
          className="answer-inspection__chevron"
          size={14}
        />
      </summary>
      <div className="answer-inspection__content">
        <dl className="answer-inspection__metrics">
          <InspectionMetric
            label={t("chat.outcome")}
            value={t(`chat.outcomes.${inspection.outcome}`)}
          />
          {snapshotId === undefined ? null : (
            <InspectionMetric
              label={t("chat.snapshotIdentity")}
              value={snapshotId}
            />
          )}
          <InspectionMetric
            label={t("chat.retrieval")}
            value={inspection.retrieval?.strategy ?? t("chat.unavailable")}
          />
          <InspectionMetric
            label={t("chat.retrievalTime")}
            value={formatMeasurement(
              inspection.measurements.retrievalMilliseconds,
              t,
            )}
          />
          <InspectionMetric
            label={t("chat.generationTime")}
            value={formatMeasurement(
              inspection.measurements.generationMilliseconds,
              t,
            )}
          />
          <InspectionMetric
            label={t("chat.totalTime")}
            value={formatMeasurement(
              inspection.measurements.totalMilliseconds,
              t,
            )}
          />
          <InspectionMetric
            label={t("chat.inputTokens")}
            value={formatMeasurement(inspection.measurements.inputTokens, t)}
          />
          <InspectionMetric
            label={t("chat.outputTokens")}
            value={formatMeasurement(inspection.measurements.outputTokens, t)}
          />
        </dl>
        <ol
          className="answer-inspection__evidence"
          aria-label={t("chat.inspectionEvidence")}
        >
          {evidence.map((reference) => (
            <li key={reference.id}>
              <button
                type="button"
                onClick={() => onReferenceSelect?.(reference)}
                disabled={reference.documentVersionId === undefined}
              >
                <strong>{reference.sourceTitle ?? reference.sourceId}</strong>
                <span>{reference.unitLocator}</span>
              </button>
              <small>
                {t("chat.retrievalRank", { rank: reference.rank })}
                {reference.cosineDistance === null ||
                reference.cosineDistance === undefined
                  ? ` - ${t("chat.unavailable")}`
                  : ` - ${t("chat.cosineDistance", { value: reference.cosineDistance.toFixed(4) })}`}
              </small>
            </li>
          ))}
        </ol>
        {inspection.graphPath?.length ? (
          <ol
            className="answer-inspection__path"
            aria-label={t("chat.graphPath")}
          >
            {inspection.graphPath.map((step) => {
              const reference = evidence.find(
                (candidate) => candidate.id === step.evidenceId,
              );
              return (
                <li key={`${step.evidenceId}:${step.relationshipType}`}>
                  <button
                    type="button"
                    disabled={reference?.documentVersionId === undefined}
                    onClick={() => {
                      if (reference) onReferenceSelect?.(reference);
                    }}
                  >
                    <strong>{step.subjectLabel}</strong>
                    <span>
                      {t(`chat.relationships.${step.relationshipType}`)}
                    </span>
                    <strong>{step.objectLabel}</strong>
                  </button>
                  <small>{step.evidenceLocator}</small>
                </li>
              );
            })}
          </ol>
        ) : null}
      </div>
    </details>
  );
}

function InspectionMetric({
  label,
  value,
}: {
  readonly label: string;
  readonly value: string;
}) {
  return (
    <div>
      <dt>{label}</dt>
      <dd>{value}</dd>
    </div>
  );
}

function formatMeasurement(
  value: number | null,
  t: ReturnType<typeof useTranslation>["t"],
): string {
  return value === null ? t("chat.unavailable") : String(value);
}

function AssistantPending() {
  const { t } = useTranslation();
  const [elapsedSeconds, setElapsedSeconds] = useState(0);

  useEffect(() => {
    const interval = window.setInterval(() => {
      setElapsedSeconds((current) => current + 1);
    }, 1_000);
    return () => window.clearInterval(interval);
  }, []);

  return (
    <output className="thinking-status" aria-live="polite">
      <span className="thinking-status__dots" aria-hidden="true">
        <i />
        <i />
        <i />
      </span>
      <span>{t("chat.thinking")}</span>
      {elapsedSeconds > 0 ? (
        <span className="thinking-status__elapsed" aria-hidden="true">
          {elapsedSeconds}s
        </span>
      ) : null}
    </output>
  );
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
          submitMode="enter"
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
