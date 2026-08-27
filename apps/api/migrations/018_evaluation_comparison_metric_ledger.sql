CREATE FUNCTION require_evaluation_run_metric_scorer_version()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    selected_policy_version text;
BEGIN
    SELECT scoring_policy_version
    INTO selected_policy_version
    FROM evaluation_run
    WHERE id = NEW.run_id
      AND corpus_id = NEW.corpus_id
    FOR KEY SHARE;

    IF NOT FOUND OR NEW.scorer_version <> selected_policy_version THEN
        RAISE EXCEPTION 'evaluation metric scorer version must match the run scoring policy' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

CREATE FUNCTION require_evaluation_terminal_case_metric_ledger()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.state NOT IN ('completed', 'abstained', 'failed', 'cancelled') THEN
        RETURN NEW;
    END IF;

    IF (
        SELECT count(*)
        FROM evaluation_run_metric
        WHERE run_id = NEW.run_id
          AND run_case_id = NEW.id
          AND corpus_id = NEW.corpus_id
    ) <> 10
       OR EXISTS (
           SELECT 1
           FROM evaluation_run_metric
           WHERE run_id = NEW.run_id
             AND run_case_id = NEW.id
             AND corpus_id = NEW.corpus_id
             AND component NOT IN (
                 'retrieval_coverage', 'citation_coverage', 'citation_validity', 'citation_scope_validity',
                 'expected_abstention_outcome', 'execution_outcome', 'latency_milliseconds', 'input_tokens',
                 'output_tokens', 'semantic_claim_support'
             )
       )
       OR EXISTS (
           SELECT component
           FROM (
               VALUES
                   ('retrieval_coverage'), ('citation_coverage'), ('citation_validity'), ('citation_scope_validity'),
                   ('expected_abstention_outcome'), ('execution_outcome'), ('latency_milliseconds'), ('input_tokens'),
                   ('output_tokens'), ('semantic_claim_support')
           ) AS required(component)
           EXCEPT
           SELECT component
           FROM evaluation_run_metric
           WHERE run_id = NEW.run_id
             AND run_case_id = NEW.id
             AND corpus_id = NEW.corpus_id
       )
       OR EXISTS (
           SELECT 1
           FROM evaluation_run_metric
           WHERE run_id = NEW.run_id
             AND run_case_id = NEW.id
             AND corpus_id = NEW.corpus_id
             AND scorer_version <> (
                 SELECT scoring_policy_version
                 FROM evaluation_run
                 WHERE id = NEW.run_id
                   AND corpus_id = NEW.corpus_id
             )
       ) THEN
        RAISE EXCEPTION 'evaluation terminal case metric ledger is incomplete or invalid' USING ERRCODE = '23514';
    END IF;

    IF NEW.state IN ('failed', 'cancelled') AND EXISTS (
        SELECT 1
        FROM evaluation_run_metric
        WHERE run_id = NEW.run_id
          AND run_case_id = NEW.id
          AND corpus_id = NEW.corpus_id
          AND metric_state = 'scored'
    ) THEN
        RAISE EXCEPTION 'failed or cancelled evaluation cases cannot have scored metrics' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

CREATE FUNCTION require_evaluation_terminal_run_metric_ledgers()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.state NOT IN ('completed', 'completed_with_failures', 'failed') THEN
        RETURN NEW;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM evaluation_run_case
        WHERE run_id = NEW.id
          AND corpus_id = NEW.corpus_id
    )
       OR EXISTS (
           SELECT 1
           FROM evaluation_run_case
           WHERE run_id = NEW.id
             AND corpus_id = NEW.corpus_id
             AND state NOT IN ('completed', 'abstained', 'failed', 'cancelled')
       )
       OR EXISTS (
           SELECT 1
           FROM evaluation_run_case AS run_case
           WHERE run_case.run_id = NEW.id
             AND run_case.corpus_id = NEW.corpus_id
             AND (
                 (
                     SELECT count(*)
                     FROM evaluation_run_metric
                     WHERE run_id = run_case.run_id
                       AND run_case_id = run_case.id
                       AND corpus_id = run_case.corpus_id
                 ) <> 10
                 OR EXISTS (
                     SELECT 1
                     FROM evaluation_run_metric
                     WHERE run_id = run_case.run_id
                       AND run_case_id = run_case.id
                       AND corpus_id = run_case.corpus_id
                       AND component NOT IN (
                           'retrieval_coverage', 'citation_coverage', 'citation_validity', 'citation_scope_validity',
                           'expected_abstention_outcome', 'execution_outcome', 'latency_milliseconds', 'input_tokens',
                           'output_tokens', 'semantic_claim_support'
                       )
                 )
                 OR EXISTS (
                     SELECT 1
                     FROM evaluation_run_metric
                     WHERE run_id = run_case.run_id
                       AND run_case_id = run_case.id
                       AND corpus_id = run_case.corpus_id
                       AND scorer_version <> NEW.scoring_policy_version
                 )
                 OR (run_case.state IN ('failed', 'cancelled') AND EXISTS (
                     SELECT 1
                     FROM evaluation_run_metric
                     WHERE run_id = run_case.run_id
                       AND run_case_id = run_case.id
                       AND corpus_id = run_case.corpus_id
                       AND metric_state = 'scored'
                 ))
             )
       ) THEN
        RAISE EXCEPTION 'terminal evaluation run requires complete valid metric ledgers' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER evaluation_run_metric_scorer_version_trigger
BEFORE INSERT ON evaluation_run_metric
FOR EACH ROW EXECUTE FUNCTION require_evaluation_run_metric_scorer_version();

CREATE TRIGGER evaluation_run_case_metric_ledger_trigger
BEFORE UPDATE ON evaluation_run_case
FOR EACH ROW EXECUTE FUNCTION require_evaluation_terminal_case_metric_ledger();

CREATE TRIGGER evaluation_run_terminal_metric_ledger_trigger
BEFORE UPDATE ON evaluation_run
FOR EACH ROW EXECUTE FUNCTION require_evaluation_terminal_run_metric_ledgers();

---- create above / drop below ----

DROP TRIGGER IF EXISTS evaluation_run_terminal_metric_ledger_trigger ON evaluation_run;
DROP TRIGGER IF EXISTS evaluation_run_case_metric_ledger_trigger ON evaluation_run_case;
DROP TRIGGER IF EXISTS evaluation_run_metric_scorer_version_trigger ON evaluation_run_metric;
DROP FUNCTION IF EXISTS require_evaluation_terminal_run_metric_ledgers();
DROP FUNCTION IF EXISTS require_evaluation_terminal_case_metric_ledger();
DROP FUNCTION IF EXISTS require_evaluation_run_metric_scorer_version();
