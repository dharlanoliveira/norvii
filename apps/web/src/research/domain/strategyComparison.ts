import type {
  ChatInspection,
  ChatReference,
  ChatStreamEvent,
  RetrievalStrategy,
} from "../../api/chat";

export const retrievalStrategies = ["vector", "graph", "hybrid"] as const;

export type StrategyComparisonState =
  | { readonly status: "idle" }
  | StrategyComparisonRunningState
  | {
      readonly status: "completed";
      readonly results: readonly StrategyComparisonResult[];
    };

export interface StrategyComparisonRunningState {
  readonly status: "running";
  readonly results: readonly StrategyComparisonResult[];
}

export interface StrategyComparisonResult {
  readonly strategy: RetrievalStrategy;
  readonly status:
    "running" | "completed" | "abstained" | "unavailable" | "failed";
  readonly answer: string | undefined;
  readonly references: readonly ChatReference[];
  readonly inspection: ChatInspection | undefined;
}

export function newStrategyComparisonResult(
  strategy: RetrievalStrategy,
): StrategyComparisonResult {
  return {
    strategy,
    status: "running",
    answer: undefined,
    references: [],
    inspection: undefined,
  };
}

export function beginStrategyComparison(): StrategyComparisonRunningState {
  return {
    status: "running",
    results: retrievalStrategies.map(newStrategyComparisonResult),
  };
}

export function applyStrategyComparisonEvent(
  result: StrategyComparisonResult,
  event: ChatStreamEvent,
  abstainedAnswer: string,
): StrategyComparisonResult {
  switch (event.type) {
    case "evidence":
      return { ...result, references: event.references };
    case "delta":
      return { ...result, answer: `${result.answer ?? ""}${event.text}` };
    case "completed":
      return {
        ...result,
        status: "completed",
        answer: event.answer,
        references: event.references,
        inspection: event.inspection,
      };
    case "abstained":
      return {
        ...result,
        status: "abstained",
        answer: abstainedAnswer,
        inspection: event.inspection,
      };
    case "cancelled":
      return { ...result, status: "failed", inspection: event.inspection };
    case "error":
      return {
        ...result,
        status: event.code === "graph_unavailable" ? "unavailable" : "failed",
        answer: event.message,
        inspection: event.inspection,
      };
    case "started":
      return result;
  }
}

export function failStrategyComparison(
  result: StrategyComparisonResult,
): StrategyComparisonResult {
  return { ...result, status: "failed" };
}

export function finishStrategyComparison(
  state: StrategyComparisonState,
): StrategyComparisonState {
  return state.status === "running" ? { ...state, status: "completed" } : state;
}
