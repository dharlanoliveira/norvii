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
import {
  type AssistantTerminalState,
  useNorviiChatRuntime,
} from "./useNorviiChatRuntime";

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
  const [strategy, setStrategy] = useState<RetrievalStrategy>("hybrid");
  const {
    runtime,
    terminalStatesByMessageId,
    referencesByMessageId,
    inspectionsByMessageId,
    lastSubmittedQuestion,
    lastSubmittedQuestionVersion,
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
        <StrategySelector strategy={strategy} onChange={setStrategy} />
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
                    terminalStatesByMessageId={terminalStatesByMessageId}
                  />
                )
              }
            </ThreadPrimitive.Messages>
            <ChatStrategyComparison
              corpusId={corpusId}
              interfaceLanguage={interfaceLanguage}
              lastSubmittedQuestion={lastSubmittedQuestion}
              lastSubmittedQuestionVersion={lastSubmittedQuestionVersion}
              onReferenceSelect={onReferenceSelect}
              provider={provider}
            />
            <ChatViewportFooter interfaceLanguage={interfaceLanguage} />
          </ThreadPrimitive.Viewport>
        </ThreadPrimitive.Root>
      </section>
    </AssistantRuntimeProvider>
  );
}

function StrategySelector({
  strategy,
  onChange,
}: {
  readonly strategy: RetrievalStrategy;
  readonly onChange: (strategy: RetrievalStrategy) => void;
}) {
  const { t } = useTranslation();
  const strategies: readonly RetrievalStrategy[] = ["vector", "hybrid"];

  return (
    <div
      aria-label={t("chat.strategy.label")}
      className="chat-strategy"
      role="radiogroup"
    >
      <span>{t("chat.strategy.label")}</span>
      <div className="chat-strategy__choices">
        {strategies.map((option) => (
          <button
            aria-checked={strategy === option}
            key={option}
            role="radio"
            type="button"
            onClick={() => onChange(option)}
          >
            {t(`chat.strategy.${option}`)}
          </button>
        ))}
      </div>
    </div>
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

function ChatStrategyComparison({
  corpusId,
  interfaceLanguage,
  lastSubmittedQuestion,
  lastSubmittedQuestionVersion,
  onReferenceSelect,
  provider,
}: {
  readonly corpusId: string;
  readonly interfaceLanguage: "en" | "pt";
  readonly lastSubmittedQuestion: string | undefined;
  readonly lastSubmittedQuestionVersion: number;
  readonly onReferenceSelect?: ((reference: ChatReference) => void) | undefined;
  readonly provider: ChatProvider;
}) {
  const isEmpty = useAuiState((state) => state.thread.isEmpty);

  if (isEmpty) {
    return null;
  }

  return (
    <StrategyComparison
      corpusId={corpusId}
      interfaceLanguage={interfaceLanguage}
      key={lastSubmittedQuestionVersion}
      onReferenceSelect={onReferenceSelect}
      provider={provider}
      question={lastSubmittedQuestion}
    />
  );
}

function ChatViewportFooter({
  interfaceLanguage,
}: {
  readonly interfaceLanguage: "en" | "pt";
}) {
  const isEmpty = useAuiState((state) => state.thread.isEmpty);

  return (
    <ThreadPrimitive.ViewportFooter
      className={`chat-viewport__footer${isEmpty ? " chat-viewport__footer--empty" : ""}`}
    >
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
    t("chat.starterQuestions.authorityReports"),
    t("chat.starterQuestions.authorityRequirements"),
    t("chat.starterQuestions.dataSubjectRights"),
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
  readonly terminalStatesByMessageId: ReadonlyMap<
    string,
    AssistantTerminalState
  >;
}

function AssistantMessage({
  referencesByMessageId,
  inspectionsByMessageId,
  onReferenceSelect,
  terminalStatesByMessageId,
}: AssistantMessageProps) {
  const { t } = useTranslation();
  const messageId = useAuiState((state) => state.message.id);
  const wasCancelled = useAuiState(
    (state) =>
      state.message.status?.type === "incomplete" &&
      state.message.status.reason === "cancelled",
  );
  const references = referencesByMessageId.get(messageId) ?? [];
  const inspection = inspectionsByMessageId.get(messageId);
  const terminalState = terminalStatesByMessageId.get(messageId);
  return (
    <MessagePrimitive.Root
      aria-label={t("chat.assistant")}
      className={`chat-message chat-message--assistant${terminalState?.kind === "error" ? " chat-message--error" : ""}`}
      role="article"
    >
      <span className="message-author">{t("chat.assistant")}</span>
      {wasCancelled || terminalState?.kind === "cancelled" ? (
        <AssistantCancelled />
      ) : terminalState?.kind === "error" ? (
        <AssistantError message={terminalState.message} />
      ) : (
        <MessagePrimitive.Parts
          components={{ Text: AssistantMarkdown, Empty: AssistantPending }}
        />
      )}
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
  readonly contribution: EvidenceContribution;
  readonly id: string;
  readonly ranks: readonly number[];
  readonly reference: ChatReference;
  readonly supportingPassageCount: number;
}

type EvidenceContribution = NonNullable<ChatReference["contribution"]>;

function CitationList({
  references,
  onReferenceSelect,
}: {
  readonly references: readonly ChatReference[];
  readonly onReferenceSelect?: ((reference: ChatReference) => void) | undefined;
}) {
  const { t } = useTranslation();
  const [selectedLocationId, setSelectedLocationId] = useState<string>();
  const citationGroups = groupCitationLocations(references);

  return (
    <section className="citation-list" aria-label={t("chat.citationLocations")}>
      <div className="citation-list__items">
        {citationGroups.map((group) => {
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
              <span className="citation-chip__metadata">
                <span className="citation-chip__contribution">
                  {t(`chat.evidenceContribution.${group.contribution}`)}
                </span>
                <span className="citation-chip__count">
                  {t("chat.supportingPassages", {
                    count: group.supportingPassageCount,
                  })}
                </span>
              </span>
            </button>
          );
        })}
      </div>
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
        contribution: reference.contribution ?? "vector",
        id: groupId,
        ranks: [reference.rank],
        reference,
        supportingPassageCount: 1,
      });
      continue;
    }
    groups.set(groupId, {
      ...existingGroup,
      contribution: combineEvidenceContribution(
        existingGroup.contribution,
        reference.contribution,
      ),
      ranks: [...existingGroup.ranks, reference.rank],
      supportingPassageCount: existingGroup.supportingPassageCount + 1,
    });
  }
  return [...groups.values()].sort(
    (left, right) => left.reference.rank - right.reference.rank,
  );
}

function combineEvidenceContribution(
  current: EvidenceContribution,
  incoming: ChatReference["contribution"],
): EvidenceContribution {
  const normalizedIncoming = incoming ?? "vector";
  if (
    current === "vector_and_graph" ||
    normalizedIncoming === "vector_and_graph" ||
    current !== normalizedIncoming
  ) {
    return "vector_and_graph";
  }
  return current;
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
        {inspection.stages?.length ? (
          <ol
            className="answer-inspection__stages"
            aria-label={t("chat.retrievalStages")}
          >
            {inspection.stages.map((stage) => (
              <li key={stage.name}>
                <strong>{t(`chat.retrievalStage.${stage.name}`)}</strong>
                <span>{t(`chat.retrievalStageState.${stage.state}`)}</span>
                <small>
                  {t("chat.retrievalStageEvidence", {
                    count: stage.evidenceCount,
                  })}
                  {stage.reasonCode
                    ? ` - ${t(`chat.retrievalStageReason.${stage.reasonCode}`)}`
                    : ""}
                </small>
              </li>
            ))}
          </ol>
        ) : null}
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
                {reference.contribution
                  ? ` - ${t(`chat.evidenceContribution.${reference.contribution}`)}`
                  : ""}
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

function AssistantError({ message }: { readonly message: string }) {
  const { t } = useTranslation();
  return (
    <div className="assistant-terminal assistant-terminal--error" role="alert">
      <strong>{t("chat.errorTitle")}</strong>
      <span>{message}</span>
    </div>
  );
}

function AssistantCancelled() {
  const { t } = useTranslation();
  return (
    <output
      className="assistant-terminal assistant-terminal--cancelled"
      role="status"
    >
      {t("chat.cancelled")}
    </output>
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
