CREATE TABLE evaluation_run (
    id uuid PRIMARY KEY,
    dataset_revision_id uuid NOT NULL,
    corpus_id uuid NOT NULL,
    snapshot_id uuid NOT NULL,
    snapshot_manifest_sha256 text NOT NULL CHECK (snapshot_manifest_sha256 ~ '^[0-9a-f]{64}$'),
    dataset_content_sha256 text NOT NULL CHECK (dataset_content_sha256 ~ '^[0-9a-f]{64}$'),
    ordered_case_set_sha256 text NOT NULL CHECK (ordered_case_set_sha256 ~ '^[0-9a-f]{64}$'),
    retrieval_strategy text NOT NULL CHECK (btrim(retrieval_strategy) <> ''),
    retrieval_configuration_fingerprint text NOT NULL CHECK (btrim(retrieval_configuration_fingerprint) <> ''),
    scoring_policy_version text NOT NULL CHECK (btrim(scoring_policy_version) <> ''),
    agent_build text NOT NULL CHECK (btrim(agent_build) <> ''),
    chat_model_identity text NOT NULL CHECK (btrim(chat_model_identity) <> ''),
    embedding_model_identity text NOT NULL CHECK (btrim(embedding_model_identity) <> ''),
    initiated_by text NOT NULL CHECK (btrim(initiated_by) <> ''),
    state text NOT NULL DEFAULT 'queued' CHECK (state IN ('queued', 'running', 'completed', 'completed_with_failures', 'failed')),
    created_at timestamptz NOT NULL DEFAULT now(),
    started_at timestamptz,
    completed_at timestamptz,
    CONSTRAINT evaluation_run_revision_ownership_fk
        FOREIGN KEY (dataset_revision_id, corpus_id, dataset_content_sha256)
        REFERENCES evaluation_dataset_revision (id, corpus_id, content_sha256),
    CONSTRAINT evaluation_run_snapshot_ownership_fk
        FOREIGN KEY (snapshot_id, corpus_id, snapshot_manifest_sha256)
        REFERENCES corpus_snapshots (id, corpus_id, manifest_sha256),
    CONSTRAINT evaluation_run_lifecycle_times_check CHECK (
        (state = 'queued' AND started_at IS NULL AND completed_at IS NULL)
        OR (state = 'running' AND started_at IS NOT NULL AND completed_at IS NULL)
        OR (state IN ('completed', 'completed_with_failures', 'failed') AND started_at IS NOT NULL AND completed_at IS NOT NULL)
    ),
    UNIQUE (id, corpus_id),
    UNIQUE (id, corpus_id, snapshot_id)
);

CREATE TABLE evaluation_run_case (
    id uuid PRIMARY KEY,
    run_id uuid NOT NULL,
    corpus_id uuid NOT NULL,
    dataset_revision_id uuid NOT NULL,
    evaluation_case_id uuid NOT NULL,
    position integer NOT NULL CHECK (position > 0),
    expected_outcome text NOT NULL CHECK (expected_outcome IN ('answer', 'abstain')),
    state text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'leased', 'completed', 'abstained', 'failed', 'cancelled')),
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    lease_token uuid,
    worker_id text,
    lease_expires_at timestamptz,
    answer text,
    graph_grounding_state text,
    safe_failure_code text,
    latency_milliseconds bigint CHECK (latency_milliseconds >= 0),
    input_tokens bigint CHECK (input_tokens >= 0),
    output_tokens bigint CHECK (output_tokens >= 0),
    started_at timestamptz,
    finished_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT evaluation_run_case_run_ownership_fk
        FOREIGN KEY (run_id, corpus_id) REFERENCES evaluation_run (id, corpus_id),
    CONSTRAINT evaluation_run_case_dataset_case_ownership_fk
        FOREIGN KEY (evaluation_case_id, corpus_id, dataset_revision_id)
        REFERENCES evaluation_dataset_case (id, corpus_id, dataset_revision_id),
    CONSTRAINT evaluation_run_case_lease_state_check CHECK (
        (state = 'leased' AND lease_token IS NOT NULL AND worker_id IS NOT NULL AND btrim(worker_id) <> '' AND lease_expires_at IS NOT NULL)
        OR (state <> 'leased' AND lease_token IS NULL AND worker_id IS NULL AND lease_expires_at IS NULL)
    ),
    CONSTRAINT evaluation_run_case_terminal_time_check CHECK (
        (state IN ('pending', 'leased') AND finished_at IS NULL)
        OR (state IN ('completed', 'abstained', 'failed', 'cancelled') AND finished_at IS NOT NULL)
    ),
    CONSTRAINT evaluation_run_case_failure_check CHECK (
        (state IN ('failed', 'cancelled') AND safe_failure_code IS NOT NULL AND btrim(safe_failure_code) <> '')
        OR (state IN ('pending', 'leased', 'completed', 'abstained') AND safe_failure_code IS NULL)
    ),
    UNIQUE (id, run_id, corpus_id),
    UNIQUE (run_id, evaluation_case_id),
    UNIQUE (run_id, position)
);

CREATE TABLE evaluation_run_expected_evidence (
    id uuid PRIMARY KEY,
    run_id uuid NOT NULL,
    run_case_id uuid NOT NULL,
    corpus_id uuid NOT NULL,
    snapshot_id uuid NOT NULL,
    source_id uuid NOT NULL,
    source_revision_id uuid NOT NULL,
    document_id uuid NOT NULL,
    legal_unit_id uuid NOT NULL,
    canonical_locator text NOT NULL CHECK (canonical_locator ~ '^[a-z][a-z-]*:[a-z0-9.-]+(/[a-z][a-z-]*:[a-z0-9.-]+)*$'),
    display_locator text NOT NULL CHECK (btrim(display_locator) <> ''),
    content_sha256 text NOT NULL CHECK (content_sha256 ~ '^[0-9a-f]{64}$'),
    ordinal integer NOT NULL CHECK (ordinal > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT evaluation_run_expected_evidence_case_ownership_fk
        FOREIGN KEY (run_case_id, run_id, corpus_id) REFERENCES evaluation_run_case (id, run_id, corpus_id),
    CONSTRAINT evaluation_run_expected_evidence_run_snapshot_fk
        FOREIGN KEY (run_id, corpus_id, snapshot_id) REFERENCES evaluation_run (id, corpus_id, snapshot_id),
    UNIQUE (run_case_id, ordinal)
);

CREATE TABLE evaluation_run_metric (
    id uuid PRIMARY KEY,
    run_id uuid NOT NULL,
    run_case_id uuid,
    corpus_id uuid NOT NULL,
    component text NOT NULL CHECK (btrim(component) <> ''),
    metric_state text NOT NULL CHECK (metric_state IN ('scored', 'not_applicable', 'not_scored', 'needs_human_review')),
    value double precision,
    numerator bigint,
    denominator bigint,
    rationale text NOT NULL CHECK (btrim(rationale) <> ''),
    scorer_version text NOT NULL CHECK (btrim(scorer_version) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT evaluation_run_metric_run_ownership_fk
        FOREIGN KEY (run_id, corpus_id) REFERENCES evaluation_run (id, corpus_id),
    CONSTRAINT evaluation_run_metric_case_ownership_fk
        FOREIGN KEY (run_case_id, run_id, corpus_id) REFERENCES evaluation_run_case (id, run_id, corpus_id),
    CONSTRAINT evaluation_run_metric_arithmetic_check CHECK (
        (metric_state = 'scored' AND value IS NOT NULL AND numerator IS NOT NULL AND denominator IS NOT NULL AND denominator > 0)
        OR (metric_state <> 'scored' AND value IS NULL AND numerator IS NULL AND denominator IS NULL)
    ),
    UNIQUE NULLS NOT DISTINCT (run_id, run_case_id, component, scorer_version)
);

CREATE TABLE evaluation_run_actual_evidence (
    id uuid PRIMARY KEY,
    run_id uuid NOT NULL,
    run_case_id uuid NOT NULL,
    corpus_id uuid NOT NULL,
    snapshot_id uuid NOT NULL,
    evidence_kind text NOT NULL CHECK (evidence_kind IN ('retrieved', 'cited')),
    position integer NOT NULL CHECK (position > 0),
    marker_position integer NOT NULL DEFAULT 0 CHECK (marker_position >= 0),
    source_id uuid NOT NULL,
    source_revision_id uuid NOT NULL,
    document_id uuid NOT NULL,
    legal_unit_id uuid NOT NULL,
    canonical_locator text NOT NULL CHECK (canonical_locator ~ '^[a-z][a-z-]*:[a-z0-9.-]+(/[a-z][a-z-]*:[a-z0-9.-]+)*$'),
    start_offset integer NOT NULL CHECK (start_offset >= 0),
    end_offset integer NOT NULL CHECK (end_offset > start_offset),
    content_sha256 text NOT NULL CHECK (content_sha256 ~ '^[0-9a-f]{64}$'),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT evaluation_run_actual_evidence_case_ownership_fk
        FOREIGN KEY (run_case_id, run_id, corpus_id) REFERENCES evaluation_run_case (id, run_id, corpus_id),
    CONSTRAINT evaluation_run_actual_evidence_run_snapshot_fk
        FOREIGN KEY (run_id, corpus_id, snapshot_id) REFERENCES evaluation_run (id, corpus_id, snapshot_id),
    CONSTRAINT evaluation_run_actual_evidence_marker_check CHECK (
        (evidence_kind = 'retrieved' AND marker_position = 0)
        OR (evidence_kind = 'cited' AND marker_position > 0)
    ),
    UNIQUE (run_case_id, evidence_kind, position)
);

CREATE INDEX evaluation_run_case_claim_idx
    ON evaluation_run_case (state, lease_expires_at, created_at, id)
    WHERE state IN ('pending', 'leased');

CREATE INDEX evaluation_run_case_run_state_idx
    ON evaluation_run_case (run_id, state, position);

CREATE FUNCTION protect_evaluation_run_identity()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.id <> OLD.id
       OR NEW.dataset_revision_id <> OLD.dataset_revision_id
       OR NEW.corpus_id <> OLD.corpus_id
       OR NEW.snapshot_id <> OLD.snapshot_id
       OR NEW.snapshot_manifest_sha256 <> OLD.snapshot_manifest_sha256
       OR NEW.dataset_content_sha256 <> OLD.dataset_content_sha256
       OR NEW.ordered_case_set_sha256 <> OLD.ordered_case_set_sha256
       OR NEW.retrieval_strategy <> OLD.retrieval_strategy
       OR NEW.retrieval_configuration_fingerprint <> OLD.retrieval_configuration_fingerprint
       OR NEW.scoring_policy_version <> OLD.scoring_policy_version
       OR NEW.agent_build <> OLD.agent_build
       OR NEW.chat_model_identity <> OLD.chat_model_identity
       OR NEW.embedding_model_identity <> OLD.embedding_model_identity
       OR NEW.initiated_by <> OLD.initiated_by THEN
        RAISE EXCEPTION 'evaluation run identity is immutable' USING ERRCODE = '55000';
    END IF;
    IF NOT ((OLD.state = 'queued' AND NEW.state = 'running')
        OR (OLD.state = 'running' AND NEW.state IN ('completed', 'completed_with_failures', 'failed'))) THEN
        RAISE EXCEPTION 'evaluation run state transition is invalid' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$;

CREATE FUNCTION protect_evaluation_run_case_terminal_result()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF OLD.state IN ('completed', 'abstained', 'failed', 'cancelled') THEN
        RAISE EXCEPTION 'evaluation terminal case result is immutable' USING ERRCODE = '55000';
    END IF;
    IF NEW.id <> OLD.id OR NEW.run_id <> OLD.run_id OR NEW.corpus_id <> OLD.corpus_id
       OR NEW.dataset_revision_id <> OLD.dataset_revision_id OR NEW.evaluation_case_id <> OLD.evaluation_case_id
       OR NEW.position <> OLD.position OR NEW.expected_outcome <> OLD.expected_outcome THEN
        RAISE EXCEPTION 'evaluation run case identity is immutable' USING ERRCODE = '55000';
    END IF;
    IF NOT ((OLD.state = 'pending' AND NEW.state = 'leased')
        OR (OLD.state = 'leased' AND NEW.state IN ('leased', 'pending', 'completed', 'abstained', 'failed', 'cancelled'))
        OR (OLD.state = 'pending' AND NEW.state = 'cancelled')) THEN
        RAISE EXCEPTION 'evaluation run case transition is invalid' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER evaluation_run_identity_trigger
BEFORE UPDATE ON evaluation_run
FOR EACH ROW EXECUTE FUNCTION protect_evaluation_run_identity();

CREATE TRIGGER evaluation_run_case_terminal_trigger
BEFORE UPDATE ON evaluation_run_case
FOR EACH ROW EXECUTE FUNCTION protect_evaluation_run_case_terminal_result();

CREATE TRIGGER evaluation_run_immutable_delete_trigger
BEFORE DELETE ON evaluation_run
FOR EACH ROW EXECUTE FUNCTION reject_evaluation_immutable_mutation();

CREATE TRIGGER evaluation_run_case_immutable_delete_trigger
BEFORE DELETE ON evaluation_run_case
FOR EACH ROW EXECUTE FUNCTION reject_evaluation_immutable_mutation();

CREATE TRIGGER evaluation_run_expected_evidence_immutable_trigger
BEFORE UPDATE OR DELETE ON evaluation_run_expected_evidence
FOR EACH ROW EXECUTE FUNCTION reject_evaluation_immutable_mutation();

CREATE TRIGGER evaluation_run_metric_immutable_trigger
BEFORE UPDATE OR DELETE ON evaluation_run_metric
FOR EACH ROW EXECUTE FUNCTION reject_evaluation_immutable_mutation();

CREATE TRIGGER evaluation_run_actual_evidence_immutable_trigger
BEFORE UPDATE OR DELETE ON evaluation_run_actual_evidence
FOR EACH ROW EXECUTE FUNCTION reject_evaluation_immutable_mutation();

---- create above / drop below ----

DROP TRIGGER IF EXISTS evaluation_run_metric_immutable_trigger ON evaluation_run_metric;
DROP TRIGGER IF EXISTS evaluation_run_actual_evidence_immutable_trigger ON evaluation_run_actual_evidence;
DROP TRIGGER IF EXISTS evaluation_run_expected_evidence_immutable_trigger ON evaluation_run_expected_evidence;
DROP TRIGGER IF EXISTS evaluation_run_case_immutable_delete_trigger ON evaluation_run_case;
DROP TRIGGER IF EXISTS evaluation_run_immutable_delete_trigger ON evaluation_run;
DROP TRIGGER IF EXISTS evaluation_run_case_terminal_trigger ON evaluation_run_case;
DROP TRIGGER IF EXISTS evaluation_run_identity_trigger ON evaluation_run;
DROP FUNCTION IF EXISTS protect_evaluation_run_case_terminal_result();
DROP FUNCTION IF EXISTS protect_evaluation_run_identity();
DROP TABLE IF EXISTS evaluation_run_metric;
DROP TABLE IF EXISTS evaluation_run_actual_evidence;
DROP TABLE IF EXISTS evaluation_run_expected_evidence;
DROP TABLE IF EXISTS evaluation_run_case;
DROP TABLE IF EXISTS evaluation_run;
