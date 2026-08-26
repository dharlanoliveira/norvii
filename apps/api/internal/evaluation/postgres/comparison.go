package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ComparisonRun reads the fixed identity, experiment variables, terminal case ledger, and stored
// metric arithmetic for one historical run. It deliberately does not join mutable dataset or
// snapshot catalog tables, so a later publication cannot alter comparison inputs.
func (repository *Repository) ComparisonRun(ctx context.Context, runID uuid.UUID) (application.ComparisonRun, error) {
	if repository == nil || repository.database == nil || runID == uuid.Nil {
		return application.ComparisonRun{}, application.ErrComparisonRunNotFound
	}
	run, err := readComparisonRunIdentity(ctx, repository.database, runID)
	if err != nil {
		return application.ComparisonRun{}, err
	}
	cases, err := readComparisonCases(ctx, repository.database, runID)
	if err != nil {
		return application.ComparisonRun{}, err
	}
	run.Cases = cases
	return run, nil
}

func readComparisonRunIdentity(ctx context.Context, database queryer, runID uuid.UUID) (application.ComparisonRun, error) {
	var run application.ComparisonRun
	err := database.QueryRow(ctx, `
		SELECT id, dataset_revision_id, dataset_content_sha256, corpus_id, snapshot_id,
		       snapshot_manifest_sha256, ordered_case_set_sha256, scoring_policy_version, state,
		       retrieval_strategy, retrieval_configuration_fingerprint, agent_build,
		       chat_model_identity, embedding_model_identity
		FROM evaluation_run
		WHERE id = $1`, runID,
	).Scan(
		&run.ID, &run.Key.DatasetRevisionID, &run.Key.DatasetContentSHA256, &run.Key.CorpusID, &run.Key.SnapshotID,
		&run.Key.SnapshotManifestSHA256, &run.Key.OrderedCaseSetSHA256, &run.Key.ScoringPolicyVersion, &run.State,
		&run.Experiment.RetrievalStrategy, &run.Experiment.RetrievalConfigurationFingerprint, &run.Experiment.AgentBuild,
		&run.Experiment.ChatModelIdentity, &run.Experiment.EmbeddingModelIdentity,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ComparisonRun{}, application.ErrComparisonRunNotFound
	}
	if err != nil {
		return application.ComparisonRun{}, fmt.Errorf("read evaluation comparison run identity: %w", err)
	}
	return run, nil
}

func readComparisonCases(ctx context.Context, database queryer, runID uuid.UUID) ([]application.ComparisonCase, error) {
	rows, err := database.Query(ctx, `
		SELECT run_case.evaluation_case_id, run_case.state, metric.component, metric.metric_state,
		       metric.scorer_version, metric.numerator, metric.denominator
		FROM evaluation_run_case AS run_case
		LEFT JOIN evaluation_run_metric AS metric
		  ON metric.run_id = run_case.run_id
		 AND metric.run_case_id = run_case.id
		 AND metric.corpus_id = run_case.corpus_id
		WHERE run_case.run_id = $1
		ORDER BY run_case.position ASC, metric.component ASC`, runID)
	if err != nil {
		return nil, fmt.Errorf("read evaluation comparison cases: %w", err)
	}
	defer rows.Close()

	casesByID := make(map[uuid.UUID]int)
	cases := make([]application.ComparisonCase, 0)
	for rows.Next() {
		var (
			datasetCaseID uuid.UUID
			state         application.RunCaseState
			name          *application.MetricName
			metricState   *application.MetricState
			scorerVersion *string
			numerator     *int64
			denominator   *int64
		)
		if err := rows.Scan(&datasetCaseID, &state, &name, &metricState, &scorerVersion, &numerator, &denominator); err != nil {
			return nil, fmt.Errorf("scan evaluation comparison case: %w", err)
		}
		caseIndex, found := casesByID[datasetCaseID]
		if !found {
			caseIndex = len(cases)
			casesByID[datasetCaseID] = caseIndex
			cases = append(cases, application.ComparisonCase{DatasetCaseID: datasetCaseID, State: state})
		}
		if name == nil || metricState == nil || scorerVersion == nil {
			if name != nil || metricState != nil || scorerVersion != nil {
				return nil, fmt.Errorf("read evaluation comparison metric ledger: incomplete metric row")
			}
			continue
		}
		cases[caseIndex].Metrics = append(cases[caseIndex].Metrics, application.ComparisonMetric{
			Name: *name, State: *metricState, ScorerVersion: *scorerVersion,
			Numerator: cloneInt64(numerator), Denominator: cloneInt64(denominator),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evaluation comparison cases: %w", err)
	}
	return cases, nil
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

var _ interface {
	ComparisonRun(context.Context, uuid.UUID) (application.ComparisonRun, error)
} = (*Repository)(nil)
