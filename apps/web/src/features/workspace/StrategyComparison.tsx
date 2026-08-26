import { BarChart3, LoaderCircle } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import type {
  ChatProvider,
  ChatReference,
  RetrievalStrategy,
} from "../../api/chat";
import type { CorpusLanguage } from "../../api/contract";
import {
  applyStrategyComparisonEvent,
  beginStrategyComparison,
  failStrategyComparison,
  finishStrategyComparison,
  newStrategyComparisonResult,
  type StrategyComparisonResult,
  type StrategyComparisonState,
} from "../../research/domain/strategyComparison";

interface StrategyComparisonProps {
  readonly corpusId: string;
  readonly interfaceLanguage: CorpusLanguage;
  readonly provider: ChatProvider;
  readonly question: string | undefined;
  readonly onReferenceSelect?: ((reference: ChatReference) => void) | undefined;
}

interface StrategyComparisonRequest {
  readonly corpusId: string;
  readonly question: string;
  readonly interfaceLanguage: CorpusLanguage;
  readonly provider: ChatProvider;
  readonly abstainedAnswer: string;
  readonly update: (result: StrategyComparisonResult) => void;
  readonly signal: AbortSignal;
}

export function StrategyComparison({
  corpusId,
  interfaceLanguage,
  provider,
  question,
  onReferenceSelect,
}: StrategyComparisonProps) {
  const { t } = useTranslation();
  const [state, setState] = useState<StrategyComparisonState>({
    status: "idle",
  });
  const activeControllers = useRef(new Set<AbortController>());

  useEffect(
    () => () => {
      activeControllers.current.forEach((controller) => controller.abort());
      activeControllers.current.clear();
    },
    [],
  );

  const run = (): void => {
    if (question === undefined || state.status === "running") return;
    const started = beginStrategyComparison();
    setState(started);
    const requests = started.results.map((result) => {
      const controller = new AbortController();
      activeControllers.current.add(controller);
      return { controller, result };
    });
    const comparison = {
      corpusId,
      question,
      interfaceLanguage,
      provider,
      abstainedAnswer: t("chat.abstained"),
      update: updateComparisonResult(setState),
    };
    void Promise.all(
      requests.map(({ controller, result }) =>
        compareStrategy(result.strategy, {
          ...comparison,
          signal: controller.signal,
        }),
      ),
    ).then(() => {
      requests.forEach(({ controller }) =>
        activeControllers.current.delete(controller),
      );
      setState(finishStrategyComparison);
    });
  };

  return (
    <section
      className="strategy-comparison"
      aria-label={t("chat.comparison.label")}
    >
      <button
        className="strategy-comparison__run"
        disabled={question === undefined || state.status === "running"}
        type="button"
        onClick={run}
      >
        {state.status === "running" ? (
          <LoaderCircle
            aria-hidden="true"
            className="strategy-comparison__spinner"
            size={14}
          />
        ) : (
          <BarChart3 aria-hidden="true" size={14} />
        )}
        {t("chat.comparison.run")}
      </button>
      {state.status !== "idle" ? (
        <div className="strategy-comparison__results">
          <p>{t("chat.comparison.question", { question: question ?? "" })}</p>
          <ol>
            {state.results.map((result) => (
              <StrategyComparisonItem
                key={result.strategy}
                onReferenceSelect={onReferenceSelect}
                result={result}
              />
            ))}
          </ol>
        </div>
      ) : null}
    </section>
  );
}

function updateComparisonResult(
  setState: (
    update: (current: StrategyComparisonState) => StrategyComparisonState,
  ) => void,
) {
  return (next: StrategyComparisonResult): void => {
    setState((current) => {
      if (current.status === "idle") return current;
      return {
        ...current,
        results: current.results.map((result) =>
          result.strategy === next.strategy ? next : result,
        ),
      };
    });
  };
}

async function compareStrategy(
  strategy: RetrievalStrategy,
  request: StrategyComparisonRequest,
): Promise<void> {
  let result = newStrategyComparisonResult(strategy);
  try {
    await request.provider.streamQuestion(
      request.corpusId,
      request.question,
      request.interfaceLanguage,
      strategy,
      request.signal,
      (event) => {
        result = applyStrategyComparisonEvent(
          result,
          event,
          request.abstainedAnswer,
        );
        request.update(result);
      },
    );
  } catch {
    result = failStrategyComparison(result);
    request.update(result);
  }
}

function StrategyComparisonItem({
  result,
  onReferenceSelect,
}: {
  readonly result: StrategyComparisonResult;
  readonly onReferenceSelect?: ((reference: ChatReference) => void) | undefined;
}) {
  const { t } = useTranslation();
  return (
    <li data-status={result.status}>
      <header>
        <strong>{t(`chat.strategy.${result.strategy}`)}</strong>
        <span>{t(`chat.comparison.status.${result.status}`)}</span>
      </header>
      {result.answer ? <p>{result.answer}</p> : null}
      {result.references.length > 0 ? (
        <div className="strategy-comparison__references">
          {result.references.map((reference) => (
            <button
              disabled={reference.documentVersionId === undefined}
              key={reference.id}
              type="button"
              onClick={() => onReferenceSelect?.(reference)}
            >
              {reference.unitLocator}
            </button>
          ))}
        </div>
      ) : null}
      {result.inspection?.assertionPath?.length ? (
        <ol className="strategy-comparison__path">
          {result.inspection.assertionPath.map((step) => {
            const evidence = result.references.find(
              (reference) => reference.unitLocator === step.evidenceLocator,
            );
            return (
              <li key={step.assertionId}>
                <button
                  disabled={evidence?.documentVersionId === undefined}
                  type="button"
                  onClick={() => {
                    if (evidence) onReferenceSelect?.(evidence);
                  }}
                >
                  {step.subjectLabel} / {t(`chat.predicates.${step.predicate}`)}{" "}
                  / {step.objectLabel}
                </button>
              </li>
            );
          })}
        </ol>
      ) : null}
      {result.inspection?.measurements.totalMilliseconds !== null &&
      result.inspection?.measurements.totalMilliseconds !== undefined ? (
        <small>
          {t("chat.comparison.totalTime", {
            value: result.inspection.measurements.totalMilliseconds,
          })}
        </small>
      ) : null}
    </li>
  );
}
