CREATE FUNCTION reject_evaluation_run_case_child_after_terminal()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    parent_state text;
BEGIN
    SELECT state
    INTO parent_state
    FROM evaluation_run_case
    WHERE id = NEW.run_case_id
      AND run_id = NEW.run_id
      AND corpus_id = NEW.corpus_id
    FOR KEY SHARE;

    IF NOT FOUND THEN
        RETURN NEW;
    END IF;

    IF parent_state NOT IN ('pending', 'leased') THEN
        RAISE EXCEPTION 'evaluation terminal case children are immutable' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$;

CREATE FUNCTION require_terminal_cases_for_evaluation_run_aggregate()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.run_case_id IS NULL AND EXISTS (
        SELECT 1
        FROM evaluation_run_case
        WHERE run_id = NEW.run_id
          AND corpus_id = NEW.corpus_id
          AND state IN ('pending', 'leased')
    ) THEN
        RAISE EXCEPTION 'evaluation run aggregates require terminal cases' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER evaluation_run_expected_evidence_terminal_child_trigger
BEFORE INSERT ON evaluation_run_expected_evidence
FOR EACH ROW EXECUTE FUNCTION reject_evaluation_run_case_child_after_terminal();

CREATE TRIGGER evaluation_run_metric_terminal_child_trigger
BEFORE INSERT ON evaluation_run_metric
FOR EACH ROW EXECUTE FUNCTION reject_evaluation_run_case_child_after_terminal();

CREATE TRIGGER evaluation_run_actual_evidence_terminal_child_trigger
BEFORE INSERT ON evaluation_run_actual_evidence
FOR EACH ROW EXECUTE FUNCTION reject_evaluation_run_case_child_after_terminal();

CREATE TRIGGER evaluation_run_metric_terminal_aggregate_trigger
BEFORE INSERT ON evaluation_run_metric
FOR EACH ROW EXECUTE FUNCTION require_terminal_cases_for_evaluation_run_aggregate();

---- create above / drop below ----

DROP TRIGGER IF EXISTS evaluation_run_metric_terminal_aggregate_trigger ON evaluation_run_metric;
DROP TRIGGER IF EXISTS evaluation_run_actual_evidence_terminal_child_trigger ON evaluation_run_actual_evidence;
DROP TRIGGER IF EXISTS evaluation_run_metric_terminal_child_trigger ON evaluation_run_metric;
DROP TRIGGER IF EXISTS evaluation_run_expected_evidence_terminal_child_trigger ON evaluation_run_expected_evidence;
DROP FUNCTION IF EXISTS require_terminal_cases_for_evaluation_run_aggregate();
DROP FUNCTION IF EXISTS reject_evaluation_run_case_child_after_terminal();
