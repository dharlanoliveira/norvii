// Package contract contains typed shapes for durable evaluation run inspection fixtures.
package contract

// EvaluationRunRevision is the immutable dataset identity persisted with an evaluation run.
type EvaluationRunRevision struct {
	ID            string `json:"id"`
	ContentSHA256 string `json:"contentSha256"`
}

// EvaluationMetric is safe stored metric arithmetic and its recorded rationale.
type EvaluationMetric struct {
	Name          string   `json:"name"`
	State         string   `json:"state"`
	Value         *float64 `json:"value"`
	Numerator     *int64   `json:"numerator"`
	Denominator   *int64   `json:"denominator"`
	Rationale     string   `json:"rationale"`
	ScorerVersion string   `json:"scorerVersion"`
}

// EvaluationRunCaseSummary is one safe, run-local status projection.
type EvaluationRunCaseSummary struct {
	ID            string  `json:"id"`
	DatasetCaseID string  `json:"datasetCaseId"`
	Position      int     `json:"position"`
	State         string  `json:"state"`
	AttemptCount  int     `json:"attemptCount"`
	FinishedAt    *string `json:"finishedAt"`
	FailureCode   string  `json:"failureCode"`
}

// EvaluationRunSummaryResponse is the immutable run-level read projection.
type EvaluationRunSummaryResponse struct {
	ID                     string                     `json:"id"`
	DatasetRevision        EvaluationRunRevision      `json:"datasetRevision"`
	CorpusID               string                     `json:"corpusId"`
	SnapshotID             string                     `json:"snapshotId"`
	SnapshotManifestSHA256 string                     `json:"snapshotManifestSha256"`
	OrderedCaseSetSHA256   string                     `json:"orderedCaseSetSha256"`
	Configuration          EvaluationConfiguration    `json:"configuration"`
	ScoringPolicyVersion   string                     `json:"scoringPolicyVersion"`
	AgentBuild             string                     `json:"agentBuild"`
	ChatModelIdentity      string                     `json:"chatModelIdentity"`
	EmbeddingModelIdentity string                     `json:"embeddingModelIdentity"`
	InitiatedBy            string                     `json:"initiatedBy"`
	State                  string                     `json:"state"`
	CreatedAt              string                     `json:"createdAt"`
	StartedAt              *string                    `json:"startedAt"`
	CompletedAt            *string                    `json:"completedAt"`
	Aggregate              EvaluationRunAggregate     `json:"aggregate"`
	Cases                  []EvaluationRunCaseSummary `json:"cases"`
}

// EvaluationConfiguration identifies the frozen retrieval selection.
type EvaluationConfiguration struct {
	Strategy    string `json:"strategy"`
	Fingerprint string `json:"fingerprint"`
}

// EvaluationRunAggregate keeps scoring and execution denominators explicit.
type EvaluationRunAggregate struct {
	Total         int64              `json:"total"`
	Eligible      int64              `json:"eligible"`
	Scored        int64              `json:"scored"`
	Failed        int64              `json:"failed"`
	Cancelled     int64              `json:"cancelled"`
	NotApplicable int64              `json:"notApplicable"`
	Metrics       []EvaluationMetric `json:"metrics"`
}

// EvaluationEvidence is immutable provenance for expected, retrieved, or cited evidence.
type EvaluationEvidence struct {
	Kind             string `json:"kind"`
	Position         int    `json:"position"`
	MarkerPosition   int    `json:"markerPosition"`
	CorpusID         string `json:"corpusId"`
	SnapshotID       string `json:"snapshotId"`
	SourceID         string `json:"sourceId"`
	SourceRevisionID string `json:"sourceRevisionId"`
	DocumentID       string `json:"documentId"`
	LegalUnitID      string `json:"legalUnitId"`
	CanonicalLocator string `json:"canonicalLocator"`
	DisplayLocator   string `json:"displayLocator"`
	StartOffset      *int   `json:"startOffset"`
	EndOffset        *int   `json:"endOffset"`
	ContentSHA256    string `json:"contentSha256"`
}

// EvaluationRunCaseResponse is one detailed immutable result with split evidence ownership.
type EvaluationRunCaseResponse struct {
	EvaluationRunCaseSummary
	RunID               string               `json:"runId"`
	CorpusID            string               `json:"corpusId"`
	SnapshotID          string               `json:"snapshotId"`
	DatasetRevisionID   string               `json:"datasetRevisionId"`
	Question            string               `json:"question"`
	ReferenceAnswer     string               `json:"referenceAnswer"`
	ExpectedOutcome     string               `json:"expectedOutcome"`
	ExpectedEvidence    []EvaluationEvidence `json:"expectedEvidence"`
	ActualEvidence      []EvaluationEvidence `json:"actualEvidence"`
	Answer              string               `json:"answer"`
	GraphGroundingState string               `json:"graphGroundingState"`
	LatencyMilliseconds *int64               `json:"latencyMilliseconds"`
	InputTokens         *int64               `json:"inputTokens"`
	OutputTokens        *int64               `json:"outputTokens"`
	Metrics             []EvaluationMetric   `json:"metrics"`
}

// EvaluationComparisonExperiment names permitted variables between compatible runs.
type EvaluationComparisonExperiment struct {
	RetrievalStrategy                 string `json:"retrievalStrategy"`
	RetrievalConfigurationFingerprint string `json:"retrievalConfigurationFingerprint"`
	AgentBuild                        string `json:"agentBuild"`
	ChatModelIdentity                 string `json:"chatModelIdentity"`
	EmbeddingModelIdentity            string `json:"embeddingModelIdentity"`
}

// EvaluationComparisonDifference identifies an immutable identity that prevents direct deltas.
type EvaluationComparisonDifference struct {
	Field string `json:"field"`
}

// EvaluationComparisonMetric is paired arithmetic for one comparable metric.
type EvaluationComparisonMetric struct {
	Name             string   `json:"name"`
	State            string   `json:"state"`
	PairedCases      int64    `json:"pairedCases"`
	LeftNumerator    int64    `json:"leftNumerator"`
	LeftDenominator  int64    `json:"leftDenominator"`
	RightNumerator   int64    `json:"rightNumerator"`
	RightDenominator int64    `json:"rightDenominator"`
	LeftValue        *float64 `json:"leftValue"`
	RightValue       *float64 `json:"rightValue"`
	Delta            *float64 `json:"delta"`
}

// EvaluationComparisonResponse describes either a strict quality comparison or a bounded mismatch.
type EvaluationComparisonResponse struct {
	ComparisonState string                           `json:"comparisonState"`
	LeftRunID       string                           `json:"leftRunId"`
	RightRunID      string                           `json:"rightRunId"`
	Left            EvaluationComparisonExperiment   `json:"left"`
	Right           EvaluationComparisonExperiment   `json:"right"`
	Differences     []EvaluationComparisonDifference `json:"differences"`
	Totals          *EvaluationComparisonTotals      `json:"totals"`
	Metrics         []EvaluationComparisonMetric     `json:"metrics"`
}

// EvaluationComparisonTotals reports paired-case and terminal-failure counts.
type EvaluationComparisonTotals struct {
	LeftCases         int64 `json:"leftCases"`
	RightCases        int64 `json:"rightCases"`
	PairedCases       int64 `json:"pairedCases"`
	LeftUnpaired      int64 `json:"leftUnpaired"`
	RightUnpaired     int64 `json:"rightUnpaired"`
	FailedOrCancelled int64 `json:"failedOrCancelled"`
	LeftFailed        int64 `json:"leftFailed"`
	LeftCancelled     int64 `json:"leftCancelled"`
	RightFailed       int64 `json:"rightFailed"`
	RightCancelled    int64 `json:"rightCancelled"`
}
