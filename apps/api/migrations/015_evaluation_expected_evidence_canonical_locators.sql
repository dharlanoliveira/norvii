ALTER TABLE evaluation_case_expected_evidence
    ADD COLUMN canonical_locator text
    CHECK (
        -- Older immutable revisions remain readable but cannot resolve through preflight.
        -- New imports require this exact canonical form at the asset/domain boundary.
        canonical_locator IS NULL
        OR canonical_locator ~ '^[a-z][a-z-]*:[a-z0-9.-]+(/[a-z][a-z-]*:[a-z0-9.-]+)*$'
    );

---- create above / drop below ----

ALTER TABLE evaluation_case_expected_evidence
    DROP COLUMN IF EXISTS canonical_locator;
