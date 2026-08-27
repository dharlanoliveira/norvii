-- Normalize the existing LGPD retrieval root into the stable evaluation corpus identity.
UPDATE corpora
SET seed_key = 'brazil-personal-data-protection',
    name = 'Brazilian Personal Data Protection (LGPD)',
    description = 'Official Brazilian federal personal-data-protection legislation.'
WHERE id = '10000000-0000-4000-8000-000000000001'
  AND seed_key = 'initial-lgpd-pt';

UPDATE sources
SET seed_key = 'brazil-personal-data-protection-lgpd'
WHERE id = '20000000-0000-4000-8000-000000000001'
  AND corpus_id = '10000000-0000-4000-8000-000000000001'
  AND seed_key = 'initial-lgpd-official-url';

INSERT INTO corpora (id, seed_key, name, description, language, jurisdiction, status)
VALUES
    (
        '10000000-0000-4000-8000-000000000003',
        'brazil-anti-corruption-white-collar-crime',
        'Brazilian Anti-Corruption and White-Collar Crime',
        'Official Brazilian federal anti-corruption, criminal, and anti-money-laundering materials.',
        'pt',
        'Brazil',
        'enabled'
    ),
    (
        '10000000-0000-4000-8000-000000000004',
        'us-fair-housing-disability-accommodations',
        'United States Fair Housing and Disability Accommodations',
        'Official United States federal fair-housing and disability-accommodation materials.',
        'en',
        'United States',
        'enabled'
    )
ON CONFLICT (seed_key) DO NOTHING;

INSERT INTO sources (id, corpus_id, seed_key, title, kind)
VALUES
    (
        '20000000-0000-4000-8000-000000000003',
        '10000000-0000-4000-8000-000000000003',
        'brazil-anti-corruption-law',
        'Brazilian Anti-Corruption Law 12,846/2013',
        'url'
    ),
    (
        '20000000-0000-4000-8000-000000000004',
        '10000000-0000-4000-8000-000000000003',
        'brazil-penal-code',
        'Brazilian Penal Code',
        'url'
    ),
    (
        '20000000-0000-4000-8000-000000000005',
        '10000000-0000-4000-8000-000000000003',
        'brazil-anti-money-laundering-law',
        'Brazilian Anti-Money-Laundering Law 9,613/1998',
        'url'
    ),
    (
        '20000000-0000-4000-8000-000000000006',
        '10000000-0000-4000-8000-000000000003',
        'coaf-resolution-36',
        'COAF Resolution 36/2021',
        'url'
    ),
    (
        '20000000-0000-4000-8000-000000000007',
        '10000000-0000-4000-8000-000000000003',
        'cgu-leniency-guidance',
        'CGU Leniency Agreement Guidance',
        'url'
    ),
    (
        '20000000-0000-4000-8000-000000000008',
        '10000000-0000-4000-8000-000000000004',
        'us-fair-housing-act-3604',
        'Fair Housing Act, 42 U.S.C. section 3604',
        'url'
    ),
    (
        '20000000-0000-4000-8000-000000000009',
        '10000000-0000-4000-8000-000000000004',
        'hud-assistance-animals',
        'HUD Assistance Animals Guidance',
        'url'
    ),
    (
        '20000000-0000-4000-8000-000000000010',
        '10000000-0000-4000-8000-000000000004',
        'hud-doj-reasonable-accommodations',
        'HUD and DOJ Reasonable Accommodations Joint Statement',
        'url'
    ),
    (
        '20000000-0000-4000-8000-000000000011',
        '10000000-0000-4000-8000-000000000004',
        'hud-report-housing-discrimination',
        'HUD Housing Discrimination Reporting Procedure',
        'url'
    ),
    (
        '20000000-0000-4000-8000-000000000012',
        '10000000-0000-4000-8000-000000000004',
        'ecfr-24-100-204',
        '24 CFR section 100.204 Reasonable Accommodations',
        'url'
    )
ON CONFLICT (seed_key) DO NOTHING;

INSERT INTO url_origins (source_id, corpus_id, submitted_url, normalized_url)
VALUES
    (
        '20000000-0000-4000-8000-000000000003',
        '10000000-0000-4000-8000-000000000003',
        'https://www.planalto.gov.br/ccivil_03/_ato2011-2014/2013/lei/l12846.htm',
        'https://www.planalto.gov.br/ccivil_03/_ato2011-2014/2013/lei/l12846.htm'
    ),
    (
        '20000000-0000-4000-8000-000000000004',
        '10000000-0000-4000-8000-000000000003',
        'https://www.planalto.gov.br/ccivil_03/decreto-lei/del2848compilado.htm',
        'https://www.planalto.gov.br/ccivil_03/decreto-lei/del2848compilado.htm'
    ),
    (
        '20000000-0000-4000-8000-000000000005',
        '10000000-0000-4000-8000-000000000003',
        'https://www.planalto.gov.br/ccivil_03/leis/l9613compilado.htm',
        'https://www.planalto.gov.br/ccivil_03/leis/l9613compilado.htm'
    ),
    (
        '20000000-0000-4000-8000-000000000006',
        '10000000-0000-4000-8000-000000000003',
        'https://www.gov.br/coaf/pt-br/acesso-a-informacao/Institucional/a-atividade-de-supervisao/regulacao/supervisao/normas-1/resolucao-coaf-no-36-de-10-de-marco-de-2021',
        'https://www.gov.br/coaf/pt-br/acesso-a-informacao/Institucional/a-atividade-de-supervisao/regulacao/supervisao/normas-1/resolucao-coaf-no-36-de-10-de-marco-de-2021'
    ),
    (
        '20000000-0000-4000-8000-000000000007',
        '10000000-0000-4000-8000-000000000003',
        'https://www.gov.br/cgu/pt-br/assuntos/integridade-privada/acordo-leniencia/acordo-de-leniencia',
        'https://www.gov.br/cgu/pt-br/assuntos/integridade-privada/acordo-leniencia/acordo-de-leniencia'
    ),
    (
        '20000000-0000-4000-8000-000000000008',
        '10000000-0000-4000-8000-000000000004',
        'https://uscode.house.gov/view.xhtml?req=%28title%3A42+section%3A3604+edition%3Aprelim%29',
        'https://uscode.house.gov/view.xhtml?req=%28title%3A42+section%3A3604+edition%3Aprelim%29'
    ),
    (
        '20000000-0000-4000-8000-000000000009',
        '10000000-0000-4000-8000-000000000004',
        'https://www.hud.gov/helping-americans/assistance-animals',
        'https://www.hud.gov/helping-americans/assistance-animals'
    ),
    (
        '20000000-0000-4000-8000-000000000010',
        '10000000-0000-4000-8000-000000000004',
        'https://www.justice.gov/sites/default/files/crt/legacy/2010/12/14/joint_statement_ra.pdf',
        'https://www.justice.gov/sites/default/files/crt/legacy/2010/12/14/joint_statement_ra.pdf'
    ),
    (
        '20000000-0000-4000-8000-000000000011',
        '10000000-0000-4000-8000-000000000004',
        'https://www.hud.gov/fairhousing/fileacomplaint',
        'https://www.hud.gov/fairhousing/fileacomplaint'
    ),
    (
        '20000000-0000-4000-8000-000000000012',
        '10000000-0000-4000-8000-000000000004',
        'https://www.ecfr.gov/current/title-24/subtitle-B/chapter-I/part-100/subpart-D/section-100.204',
        'https://www.ecfr.gov/current/title-24/subtitle-B/chapter-I/part-100/subpart-D/section-100.204'
    )
ON CONFLICT (source_id) DO NOTHING;

ALTER TABLE corpus_snapshots
    ADD CONSTRAINT corpus_snapshots_id_corpus_manifest_key
    UNIQUE (id, corpus_id, manifest_sha256);

CREATE TABLE evaluation_dataset_revision (
    id uuid PRIMARY KEY,
    corpus_id uuid NOT NULL REFERENCES corpora (id),
    dataset_key text NOT NULL CHECK (btrim(dataset_key) <> ''),
    semantic_revision text NOT NULL CHECK (btrim(semantic_revision) <> ''),
    jurisdiction text NOT NULL CHECK (btrim(jurisdiction) <> ''),
    manifest_sha256 text NOT NULL CHECK (manifest_sha256 ~ '^[0-9a-f]{64}$'),
    jsonl_sha256 text NOT NULL CHECK (jsonl_sha256 ~ '^[0-9a-f]{64}$'),
    content_sha256 text NOT NULL CHECK (content_sha256 ~ '^[0-9a-f]{64}$'),
    manifest_path text NOT NULL CHECK (
        btrim(manifest_path) <> ''
        AND manifest_path !~ '^/'
        AND manifest_path !~ '(^|/)\.\.(/|$)'
    ),
    jsonl_path text NOT NULL CHECK (
        btrim(jsonl_path) <> ''
        AND jsonl_path !~ '^/'
        AND jsonl_path !~ '(^|/)\.\.(/|$)'
    ),
    declared_snapshot_date date NOT NULL,
    query_languages text[] NOT NULL CHECK (
        cardinality(query_languages) > 0
        AND query_languages <@ ARRAY['en', 'pt']::text[]
    ),
    authoritative_evidence_language text NOT NULL CHECK (
        authoritative_evidence_language IN ('en', 'pt-BR')
    ),
    importer_version text NOT NULL CHECK (btrim(importer_version) <> ''),
    import_diagnostics jsonb NOT NULL DEFAULT '[]'::jsonb
        CHECK (jsonb_typeof(import_diagnostics) = 'array' AND octet_length(import_diagnostics::text) <= 8192),
    imported_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (id, corpus_id),
    UNIQUE (id, corpus_id, content_sha256),
    UNIQUE (corpus_id, dataset_key, semantic_revision),
    UNIQUE (corpus_id, content_sha256)
);

CREATE TABLE evaluation_dataset_source (
    id uuid PRIMARY KEY,
    dataset_revision_id uuid NOT NULL,
    corpus_id uuid NOT NULL,
    source_alias text NOT NULL CHECK (btrim(source_alias) <> ''),
    title text NOT NULL CHECK (btrim(title) <> ''),
    official_url text NOT NULL CHECK (official_url ~ '^https://'),
    issuing_authority text NOT NULL CHECK (btrim(issuing_authority) <> ''),
    document_type text NOT NULL CHECK (btrim(document_type) <> ''),
    authority_role text NOT NULL CHECK (authority_role IN ('statute', 'regulation', 'guidance', 'procedure')),
    corpus_source_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT evaluation_dataset_source_revision_ownership_fk
        FOREIGN KEY (dataset_revision_id, corpus_id)
        REFERENCES evaluation_dataset_revision (id, corpus_id),
    CONSTRAINT evaluation_dataset_source_corpus_source_ownership_fk
        FOREIGN KEY (corpus_id, corpus_source_id)
        REFERENCES sources (corpus_id, id),
    UNIQUE (id, corpus_id),
    UNIQUE (id, corpus_id, dataset_revision_id),
    UNIQUE (corpus_id, dataset_revision_id, source_alias)
);

CREATE TABLE evaluation_dataset_case (
    id uuid PRIMARY KEY,
    dataset_revision_id uuid NOT NULL,
    corpus_id uuid NOT NULL,
    position integer NOT NULL CHECK (position > 0),
    external_case_id text NOT NULL CHECK (btrim(external_case_id) <> ''),
    query_language text NOT NULL CHECK (query_language IN ('en', 'pt')),
    asset_language text NOT NULL CHECK (asset_language IN ('en', 'pt-BR')),
    question text NOT NULL CHECK (btrim(question) <> ''),
    reference_answer text NOT NULL CHECK (btrim(reference_answer) <> ''),
    category text NOT NULL CHECK (btrim(category) <> ''),
    authoritative_evidence_language text NOT NULL CHECK (
        authoritative_evidence_language IN ('en', 'pt-BR')
    ),
    expected_outcome text NOT NULL DEFAULT 'answer'
        CONSTRAINT evaluation_dataset_case_expected_outcome_value_check
        CHECK (expected_outcome IN ('answer', 'abstain')),
    expected_reason_code text,
    reciprocal_case_external_id text NOT NULL CHECK (btrim(reciprocal_case_external_id) <> ''),
    case_checksum text NOT NULL CHECK (case_checksum ~ '^[0-9a-f]{64}$'),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT evaluation_dataset_case_revision_ownership_fk
        FOREIGN KEY (dataset_revision_id, corpus_id)
        REFERENCES evaluation_dataset_revision (id, corpus_id),
    CONSTRAINT evaluation_dataset_case_reciprocal_reference_fk
        FOREIGN KEY (dataset_revision_id, reciprocal_case_external_id)
        REFERENCES evaluation_dataset_case (dataset_revision_id, external_case_id)
        DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT evaluation_dataset_case_language_pair_check CHECK (
        (query_language = 'en' AND asset_language = 'en')
        OR (query_language = 'pt' AND asset_language = 'pt-BR')
    ),
    CONSTRAINT evaluation_dataset_case_expected_outcome_reason_check CHECK (
        (expected_outcome = 'answer' AND expected_reason_code IS NULL)
        OR (expected_outcome = 'abstain' AND expected_reason_code IS NOT NULL AND btrim(expected_reason_code) <> '')
    ),
    CONSTRAINT evaluation_dataset_case_distinct_reciprocal_check
        CHECK (external_case_id <> reciprocal_case_external_id),
    UNIQUE (dataset_revision_id, external_case_id),
    UNIQUE (dataset_revision_id, position),
    UNIQUE (id, corpus_id, dataset_revision_id),
    UNIQUE (id, corpus_id, dataset_revision_id, query_language, case_checksum)
);

CREATE TABLE evaluation_case_expected_evidence (
    id uuid PRIMARY KEY,
    dataset_revision_id uuid NOT NULL,
    corpus_id uuid NOT NULL,
    evaluation_case_id uuid NOT NULL,
    source_alias text NOT NULL CHECK (btrim(source_alias) <> ''),
    ordinal integer NOT NULL CHECK (ordinal > 0),
    display_locator text NOT NULL CHECK (btrim(display_locator) <> ''),
    required_propositions jsonb NOT NULL CHECK (
        jsonb_typeof(required_propositions) = 'array'
        AND jsonb_array_length(required_propositions) > 0
    ),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT evaluation_case_expected_evidence_case_ownership_fk
        FOREIGN KEY (evaluation_case_id, corpus_id, dataset_revision_id)
        REFERENCES evaluation_dataset_case (id, corpus_id, dataset_revision_id),
    CONSTRAINT evaluation_case_expected_evidence_source_ownership_fk
        FOREIGN KEY (corpus_id, dataset_revision_id, source_alias)
        REFERENCES evaluation_dataset_source (corpus_id, dataset_revision_id, source_alias),
    UNIQUE (evaluation_case_id, ordinal)
);

CREATE TABLE evaluation_dataset_starter_case (
    id uuid PRIMARY KEY,
    dataset_revision_id uuid NOT NULL,
    corpus_id uuid NOT NULL,
    evaluation_case_id uuid NOT NULL,
    rank integer NOT NULL CHECK (rank BETWEEN 1 AND 5),
    query_language text NOT NULL CHECK (query_language IN ('en', 'pt')),
    case_checksum text NOT NULL CHECK (case_checksum ~ '^[0-9a-f]{64}$'),
    is_review_eligible boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT evaluation_dataset_starter_case_case_ownership_fk
        FOREIGN KEY (evaluation_case_id, corpus_id, dataset_revision_id, query_language, case_checksum)
        REFERENCES evaluation_dataset_case (id, corpus_id, dataset_revision_id, query_language, case_checksum),
    UNIQUE (dataset_revision_id, evaluation_case_id),
    UNIQUE (dataset_revision_id, rank, query_language)
);

CREATE TABLE evaluation_dataset_publication (
    id uuid PRIMARY KEY,
    dataset_revision_id uuid NOT NULL,
    corpus_id uuid NOT NULL,
    review_decision text NOT NULL CHECK (review_decision IN ('pending', 'approved', 'rejected')),
    reviewer_identity text NOT NULL CHECK (btrim(reviewer_identity) <> ''),
    review_note text NOT NULL DEFAULT '' CHECK (char_length(review_note) <= 2000),
    publication_state text NOT NULL CHECK (publication_state IN ('draft', 'available', 'superseded', 'withdrawn')),
    reviewed_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT evaluation_dataset_publication_revision_ownership_fk
        FOREIGN KEY (dataset_revision_id, corpus_id)
        REFERENCES evaluation_dataset_revision (id, corpus_id),
    CONSTRAINT evaluation_dataset_publication_available_review_check CHECK (
        publication_state <> 'available' OR review_decision = 'approved'
    ),
    UNIQUE (id, corpus_id, dataset_revision_id)
);

CREATE TABLE corpus_opening_suggestion_set (
    id uuid PRIMARY KEY,
    corpus_id uuid NOT NULL,
    snapshot_id uuid NOT NULL,
    snapshot_manifest_sha256 text NOT NULL CHECK (snapshot_manifest_sha256 ~ '^[0-9a-f]{64}$'),
    dataset_revision_id uuid NOT NULL,
    source_dataset_content_sha256 text NOT NULL CHECK (source_dataset_content_sha256 ~ '^[0-9a-f]{64}$'),
    selection_policy_version text NOT NULL CHECK (btrim(selection_policy_version) <> ''),
    published_by text NOT NULL CHECK (btrim(published_by) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT corpus_opening_suggestion_set_snapshot_ownership_fk
        FOREIGN KEY (snapshot_id, corpus_id, snapshot_manifest_sha256)
        REFERENCES corpus_snapshots (id, corpus_id, manifest_sha256),
    CONSTRAINT corpus_opening_suggestion_set_revision_ownership_fk
        FOREIGN KEY (dataset_revision_id, corpus_id, source_dataset_content_sha256)
        REFERENCES evaluation_dataset_revision (id, corpus_id, content_sha256),
    UNIQUE (id, corpus_id),
    UNIQUE (id, corpus_id, dataset_revision_id),
    UNIQUE (corpus_id, snapshot_id, source_dataset_content_sha256, selection_policy_version)
);

CREATE TABLE corpus_opening_suggestion_item (
    id uuid PRIMARY KEY,
    suggestion_set_id uuid NOT NULL,
    corpus_id uuid NOT NULL,
    dataset_revision_id uuid NOT NULL,
    rank integer NOT NULL CHECK (rank BETWEEN 1 AND 5),
    evaluation_case_id uuid NOT NULL,
    case_checksum text NOT NULL CHECK (case_checksum ~ '^[0-9a-f]{64}$'),
    query_language text NOT NULL CHECK (query_language IN ('en', 'pt')),
    question text NOT NULL CHECK (btrim(question) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT corpus_opening_suggestion_item_set_ownership_fk
        FOREIGN KEY (suggestion_set_id, corpus_id, dataset_revision_id)
        REFERENCES corpus_opening_suggestion_set (id, corpus_id, dataset_revision_id),
    CONSTRAINT corpus_opening_suggestion_item_case_ownership_fk
        FOREIGN KEY (evaluation_case_id, corpus_id, dataset_revision_id, query_language, case_checksum)
        REFERENCES evaluation_dataset_case (id, corpus_id, dataset_revision_id, query_language, case_checksum),
    UNIQUE (suggestion_set_id, evaluation_case_id),
    UNIQUE (suggestion_set_id, query_language, rank)
);

CREATE INDEX evaluation_dataset_revision_corpus_imported_idx
    ON evaluation_dataset_revision (corpus_id, imported_at DESC, id DESC);

CREATE INDEX evaluation_dataset_source_binding_idx
    ON evaluation_dataset_source (corpus_id, corpus_source_id)
    WHERE corpus_source_id IS NOT NULL;

CREATE INDEX evaluation_dataset_case_revision_position_idx
    ON evaluation_dataset_case (dataset_revision_id, position, id);

CREATE INDEX evaluation_case_expected_evidence_case_idx
    ON evaluation_case_expected_evidence (evaluation_case_id, ordinal);

CREATE INDEX evaluation_dataset_publication_revision_latest_idx
    ON evaluation_dataset_publication (dataset_revision_id, reviewed_at DESC, id DESC);

CREATE INDEX corpus_opening_suggestion_set_snapshot_idx
    ON corpus_opening_suggestion_set (corpus_id, snapshot_id, created_at DESC, id DESC);

CREATE INDEX corpus_opening_suggestion_item_rank_idx
    ON corpus_opening_suggestion_item (suggestion_set_id, query_language, rank);

CREATE FUNCTION reject_evaluation_immutable_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'evaluation catalog and suggestion rows are append-only'
        USING ERRCODE = '55000';
END;
$$;

CREATE FUNCTION lock_evaluation_dataset_source_binding()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF OLD.corpus_source_id IS NULL
       AND NEW.corpus_source_id IS NOT NULL
       AND NEW.id = OLD.id
       AND NEW.dataset_revision_id = OLD.dataset_revision_id
       AND NEW.corpus_id = OLD.corpus_id
       AND NEW.source_alias = OLD.source_alias
       AND NEW.title = OLD.title
       AND NEW.official_url = OLD.official_url
       AND NEW.issuing_authority = OLD.issuing_authority
       AND NEW.document_type = OLD.document_type
       AND NEW.authority_role = OLD.authority_role
       AND NEW.created_at = OLD.created_at THEN
        RETURN NEW;
    END IF;

    RAISE EXCEPTION 'evaluation dataset source requirements are immutable after creation'
        USING ERRCODE = '55000';
END;
$$;

CREATE FUNCTION validate_evaluation_dataset_case_pair()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    reciprocal_case evaluation_dataset_case%ROWTYPE;
BEGIN
    SELECT *
    INTO reciprocal_case
    FROM evaluation_dataset_case
    WHERE dataset_revision_id = NEW.dataset_revision_id
      AND external_case_id = NEW.reciprocal_case_external_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'evaluation case % is missing its reciprocal case %',
            NEW.external_case_id,
            NEW.reciprocal_case_external_id
            USING ERRCODE = '23514';
    END IF;

    IF reciprocal_case.corpus_id <> NEW.corpus_id
       OR reciprocal_case.reciprocal_case_external_id <> NEW.external_case_id
       OR reciprocal_case.query_language = NEW.query_language THEN
        RAISE EXCEPTION 'evaluation case % has an invalid reciprocal pair', NEW.external_case_id
            USING ERRCODE = '23514';
    END IF;

    RETURN NULL;
END;
$$;

CREATE FUNCTION validate_evaluation_dataset_starter_pair()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    paired_case_id uuid;
    paired_language text;
BEGIN
    IF NOT NEW.is_review_eligible THEN
        RAISE EXCEPTION 'evaluation starter case selections must be review eligible'
            USING ERRCODE = '23514';
    END IF;

    SELECT paired.id, paired.query_language
    INTO paired_case_id, paired_language
    FROM evaluation_dataset_case AS selected
    JOIN evaluation_dataset_case AS paired
      ON paired.dataset_revision_id = selected.dataset_revision_id
     AND paired.external_case_id = selected.reciprocal_case_external_id
    WHERE selected.id = NEW.evaluation_case_id
      AND selected.dataset_revision_id = NEW.dataset_revision_id
      AND selected.corpus_id = NEW.corpus_id;

    IF NOT FOUND OR NOT EXISTS (
        SELECT 1
        FROM evaluation_dataset_starter_case AS paired_selection
        WHERE paired_selection.dataset_revision_id = NEW.dataset_revision_id
          AND paired_selection.corpus_id = NEW.corpus_id
          AND paired_selection.evaluation_case_id = paired_case_id
          AND paired_selection.rank = NEW.rank
          AND paired_selection.query_language = paired_language
          AND paired_selection.is_review_eligible
    ) THEN
        RAISE EXCEPTION 'evaluation starter case rank % requires its reciprocal reviewed pair', NEW.rank
            USING ERRCODE = '23514';
    END IF;

    RETURN NULL;
END;
$$;

CREATE FUNCTION validate_evaluation_dataset_publication()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.publication_state <> 'available' THEN
        RETURN NULL;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM evaluation_dataset_source
        WHERE dataset_revision_id = NEW.dataset_revision_id
          AND corpus_id = NEW.corpus_id
          AND corpus_source_id IS NULL
    ) THEN
        RAISE EXCEPTION 'available evaluation datasets require every source binding'
            USING ERRCODE = '23514';
    END IF;

    IF (
        SELECT count(*)
        FROM evaluation_dataset_starter_case
        WHERE dataset_revision_id = NEW.dataset_revision_id
          AND corpus_id = NEW.corpus_id
          AND is_review_eligible
    ) <> 10 OR EXISTS (
        SELECT 1
        FROM generate_series(1, 5) AS expected_rank(rank)
        WHERE (
            SELECT count(*)
            FROM evaluation_dataset_starter_case
            WHERE dataset_revision_id = NEW.dataset_revision_id
              AND corpus_id = NEW.corpus_id
              AND rank = expected_rank.rank
              AND is_review_eligible
        ) <> 2
    ) THEN
        RAISE EXCEPTION 'available evaluation datasets require five complete reviewed starter pairs'
            USING ERRCODE = '23514';
    END IF;

    RETURN NULL;
END;
$$;

CREATE FUNCTION validate_corpus_opening_suggestion_item()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM evaluation_dataset_starter_case
        WHERE dataset_revision_id = NEW.dataset_revision_id
          AND corpus_id = NEW.corpus_id
          AND evaluation_case_id = NEW.evaluation_case_id
          AND rank = NEW.rank
          AND query_language = NEW.query_language
          AND case_checksum = NEW.case_checksum
          AND is_review_eligible
    ) THEN
        RAISE EXCEPTION 'opening suggestion items must reference a reviewed starter selection'
            USING ERRCODE = '23514';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM evaluation_dataset_case
        WHERE id = NEW.evaluation_case_id
          AND corpus_id = NEW.corpus_id
          AND dataset_revision_id = NEW.dataset_revision_id
          AND query_language = NEW.query_language
          AND case_checksum = NEW.case_checksum
          AND question = NEW.question
    ) THEN
        RAISE EXCEPTION 'opening suggestion item question must equal its immutable evaluation case'
            USING ERRCODE = '23514';
    END IF;

    RETURN NULL;
END;
$$;

CREATE FUNCTION validate_corpus_opening_suggestion_set()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF (
        SELECT publication_state
        FROM evaluation_dataset_publication
        WHERE dataset_revision_id = NEW.dataset_revision_id
          AND corpus_id = NEW.corpus_id
        ORDER BY reviewed_at DESC, id DESC
        LIMIT 1
    ) IS DISTINCT FROM 'available' THEN
        RAISE EXCEPTION 'opening suggestion sets require an available dataset revision'
            USING ERRCODE = '23514';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM evaluation_dataset_source AS requirement
        LEFT JOIN corpus_snapshot_documents AS member
          ON member.snapshot_id = NEW.snapshot_id
         AND member.corpus_id = NEW.corpus_id
         AND member.source_id = requirement.corpus_source_id
        WHERE requirement.dataset_revision_id = NEW.dataset_revision_id
          AND requirement.corpus_id = NEW.corpus_id
          AND (requirement.corpus_source_id IS NULL OR member.source_id IS NULL)
    ) THEN
        RAISE EXCEPTION 'opening suggestion sets require every bound source in their snapshot'
            USING ERRCODE = '23514';
    END IF;

    IF (
        SELECT count(*)
        FROM corpus_opening_suggestion_item
        WHERE suggestion_set_id = NEW.id
    ) <> 10 OR EXISTS (
        SELECT 1
        FROM generate_series(1, 5) AS expected_rank(rank)
        CROSS JOIN (VALUES ('en'::text), ('pt'::text)) AS expected_language(query_language)
        WHERE NOT EXISTS (
            SELECT 1
            FROM corpus_opening_suggestion_item
            WHERE suggestion_set_id = NEW.id
              AND rank = expected_rank.rank
              AND query_language = expected_language.query_language
        )
    ) THEN
        RAISE EXCEPTION 'opening suggestion sets require five complete language pairs'
            USING ERRCODE = '23514';
    END IF;

    RETURN NULL;
END;
$$;

CREATE TRIGGER evaluation_dataset_revision_immutable_trigger
BEFORE UPDATE OR DELETE ON evaluation_dataset_revision
FOR EACH ROW EXECUTE FUNCTION reject_evaluation_immutable_mutation();

CREATE TRIGGER evaluation_dataset_source_binding_lock_trigger
BEFORE UPDATE OR DELETE ON evaluation_dataset_source
FOR EACH ROW EXECUTE FUNCTION lock_evaluation_dataset_source_binding();

CREATE TRIGGER evaluation_dataset_case_immutable_trigger
BEFORE UPDATE OR DELETE ON evaluation_dataset_case
FOR EACH ROW EXECUTE FUNCTION reject_evaluation_immutable_mutation();

CREATE TRIGGER evaluation_case_expected_evidence_immutable_trigger
BEFORE UPDATE OR DELETE ON evaluation_case_expected_evidence
FOR EACH ROW EXECUTE FUNCTION reject_evaluation_immutable_mutation();

CREATE TRIGGER evaluation_dataset_starter_case_immutable_trigger
BEFORE UPDATE OR DELETE ON evaluation_dataset_starter_case
FOR EACH ROW EXECUTE FUNCTION reject_evaluation_immutable_mutation();

CREATE TRIGGER evaluation_dataset_publication_immutable_trigger
BEFORE UPDATE OR DELETE ON evaluation_dataset_publication
FOR EACH ROW EXECUTE FUNCTION reject_evaluation_immutable_mutation();

CREATE TRIGGER corpus_opening_suggestion_set_immutable_trigger
BEFORE UPDATE OR DELETE ON corpus_opening_suggestion_set
FOR EACH ROW EXECUTE FUNCTION reject_evaluation_immutable_mutation();

CREATE TRIGGER corpus_opening_suggestion_item_immutable_trigger
BEFORE UPDATE OR DELETE ON corpus_opening_suggestion_item
FOR EACH ROW EXECUTE FUNCTION reject_evaluation_immutable_mutation();

CREATE CONSTRAINT TRIGGER evaluation_dataset_case_reciprocal_pair_trigger
AFTER INSERT ON evaluation_dataset_case
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_evaluation_dataset_case_pair();

CREATE CONSTRAINT TRIGGER evaluation_dataset_starter_case_pair_trigger
AFTER INSERT ON evaluation_dataset_starter_case
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_evaluation_dataset_starter_pair();

CREATE CONSTRAINT TRIGGER evaluation_dataset_publication_lifecycle_trigger
AFTER INSERT ON evaluation_dataset_publication
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_evaluation_dataset_publication();

CREATE CONSTRAINT TRIGGER corpus_opening_suggestion_item_validation_trigger
AFTER INSERT ON corpus_opening_suggestion_item
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_corpus_opening_suggestion_item();

CREATE CONSTRAINT TRIGGER corpus_opening_suggestion_set_validation_trigger
AFTER INSERT ON corpus_opening_suggestion_set
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_corpus_opening_suggestion_set();

---- create above / drop below ----

DROP TABLE IF EXISTS corpus_opening_suggestion_item;
DROP TABLE IF EXISTS corpus_opening_suggestion_set;
DROP TABLE IF EXISTS evaluation_dataset_publication;
DROP TABLE IF EXISTS evaluation_dataset_starter_case;
DROP TABLE IF EXISTS evaluation_case_expected_evidence;
DROP TABLE IF EXISTS evaluation_dataset_case;
DROP TABLE IF EXISTS evaluation_dataset_source;
DROP TABLE IF EXISTS evaluation_dataset_revision;

DROP FUNCTION IF EXISTS validate_corpus_opening_suggestion_set();
DROP FUNCTION IF EXISTS validate_corpus_opening_suggestion_item();
DROP FUNCTION IF EXISTS validate_evaluation_dataset_publication();
DROP FUNCTION IF EXISTS validate_evaluation_dataset_starter_pair();
DROP FUNCTION IF EXISTS validate_evaluation_dataset_case_pair();
DROP FUNCTION IF EXISTS lock_evaluation_dataset_source_binding();
DROP FUNCTION IF EXISTS reject_evaluation_immutable_mutation();

ALTER TABLE corpus_snapshots
    DROP CONSTRAINT IF EXISTS corpus_snapshots_id_corpus_manifest_key;

DELETE FROM url_origins
WHERE source_id IN (
    '20000000-0000-4000-8000-000000000003',
    '20000000-0000-4000-8000-000000000004',
    '20000000-0000-4000-8000-000000000005',
    '20000000-0000-4000-8000-000000000006',
    '20000000-0000-4000-8000-000000000007',
    '20000000-0000-4000-8000-000000000008',
    '20000000-0000-4000-8000-000000000009',
    '20000000-0000-4000-8000-000000000010',
    '20000000-0000-4000-8000-000000000011',
    '20000000-0000-4000-8000-000000000012'
);

DELETE FROM sources
WHERE id IN (
    '20000000-0000-4000-8000-000000000003',
    '20000000-0000-4000-8000-000000000004',
    '20000000-0000-4000-8000-000000000005',
    '20000000-0000-4000-8000-000000000006',
    '20000000-0000-4000-8000-000000000007',
    '20000000-0000-4000-8000-000000000008',
    '20000000-0000-4000-8000-000000000009',
    '20000000-0000-4000-8000-000000000010',
    '20000000-0000-4000-8000-000000000011',
    '20000000-0000-4000-8000-000000000012'
);

DELETE FROM corpora
WHERE id IN (
    '10000000-0000-4000-8000-000000000003',
    '10000000-0000-4000-8000-000000000004'
);

UPDATE sources
SET seed_key = 'initial-lgpd-official-url'
WHERE id = '20000000-0000-4000-8000-000000000001'
  AND corpus_id = '10000000-0000-4000-8000-000000000001'
  AND seed_key = 'brazil-personal-data-protection-lgpd';

UPDATE corpora
SET seed_key = 'initial-lgpd-pt',
    name = 'Brazilian General Data Protection Law',
    description = 'Official Brazilian federal data-protection legislation.'
WHERE id = '10000000-0000-4000-8000-000000000001'
  AND seed_key = 'brazil-personal-data-protection';
