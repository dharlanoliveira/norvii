CREATE TABLE corpora (
    id uuid PRIMARY KEY,
    seed_key text UNIQUE,
    name text NOT NULL CHECK (btrim(name) <> ''),
    description text NOT NULL CHECK (btrim(description) <> ''),
    language text NOT NULL CHECK (language IN ('en', 'pt')),
    jurisdiction text NOT NULL CHECK (btrim(jurisdiction) <> ''),
    status text NOT NULL DEFAULT 'enabled' CHECK (status IN ('enabled', 'disabled')),
    version integer NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (id, language)
);

CREATE INDEX corpora_enabled_order_idx
    ON corpora (language, name, id)
    WHERE status = 'enabled';

CREATE TABLE sources (
    id uuid PRIMARY KEY,
    corpus_id uuid NOT NULL,
    seed_key text UNIQUE,
    title text NOT NULL CHECK (btrim(title) <> ''),
    kind text NOT NULL CHECK (kind IN ('pdf', 'url')),
    processing_status text NOT NULL DEFAULT 'pending'
        CHECK (processing_status IN ('pending', 'processing', 'ready', 'failed')),
    latest_failure_category text,
    latest_ready_document_id uuid,
    version integer NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sources_corpus_ownership_fk
        FOREIGN KEY (corpus_id) REFERENCES corpora (id),
    UNIQUE (corpus_id, id)
);

CREATE INDEX sources_corpus_order_idx ON sources (corpus_id, created_at, id);

CREATE TABLE pdf_origins (
    source_id uuid PRIMARY KEY,
    corpus_id uuid NOT NULL,
    original_filename text NOT NULL CHECK (btrim(original_filename) <> ''),
    delivery_filename text NOT NULL CHECK (btrim(delivery_filename) <> ''),
    declared_media_type text NOT NULL,
    detected_media_type text NOT NULL CHECK (detected_media_type = 'application/pdf'),
    byte_size bigint NOT NULL CHECK (byte_size > 0 AND byte_size <= 10485760),
    sha256 text NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    content bytea NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pdf_origins_source_ownership_fk
        FOREIGN KEY (corpus_id, source_id) REFERENCES sources (corpus_id, id),
    CONSTRAINT pdf_origins_content_size_check CHECK (octet_length(content) = byte_size),
    UNIQUE (corpus_id, sha256)
);

CREATE TABLE url_origins (
    source_id uuid PRIMARY KEY,
    corpus_id uuid NOT NULL,
    submitted_url text NOT NULL CHECK (submitted_url ~ '^https://'),
    normalized_url text NOT NULL CHECK (normalized_url ~ '^https://'),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT url_origins_source_ownership_fk
        FOREIGN KEY (corpus_id, source_id) REFERENCES sources (corpus_id, id),
    UNIQUE (corpus_id, normalized_url)
);

CREATE TABLE ingestion_work (
    id uuid PRIMARY KEY,
    source_id uuid NOT NULL,
    corpus_id uuid NOT NULL,
    reason text NOT NULL CHECK (reason IN ('initial', 'retry', 'reprocess')),
    status text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'leased', 'succeeded', 'failed')),
    requested_at timestamptz NOT NULL DEFAULT now(),
    lease_token uuid,
    worker_id text,
    lease_expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ingestion_work_source_ownership_fk
        FOREIGN KEY (corpus_id, source_id) REFERENCES sources (corpus_id, id),
    CONSTRAINT ingestion_work_lease_state_check CHECK (
        (status = 'leased' AND lease_token IS NOT NULL AND worker_id IS NOT NULL AND lease_expires_at IS NOT NULL)
        OR
        (status <> 'leased' AND lease_token IS NULL AND worker_id IS NULL AND lease_expires_at IS NULL)
    ),
    UNIQUE (corpus_id, source_id, id)
);

CREATE UNIQUE INDEX ingestion_work_active_source_uidx
    ON ingestion_work (source_id)
    WHERE status IN ('pending', 'leased');

CREATE INDEX ingestion_work_pending_order_idx
    ON ingestion_work (requested_at, id)
    WHERE status = 'pending';

CREATE TABLE processing_attempts (
    id uuid PRIMARY KEY,
    work_id uuid NOT NULL,
    source_id uuid NOT NULL,
    corpus_id uuid NOT NULL,
    attempt_number integer NOT NULL CHECK (attempt_number > 0),
    pipeline_version text NOT NULL CHECK (btrim(pipeline_version) <> ''),
    status text NOT NULL CHECK (status IN ('processing', 'succeeded', 'failed')),
    lease_token uuid NOT NULL,
    worker_id text NOT NULL CHECK (btrim(worker_id) <> ''),
    started_at timestamptz NOT NULL,
    finished_at timestamptz,
    failure_category text,
    failure_detail text CHECK (char_length(failure_detail) <= 500),
    acquired_byte_count bigint CHECK (acquired_byte_count >= 0),
    normalized_character_count bigint CHECK (normalized_character_count >= 0),
    unit_count integer CHECK (unit_count >= 0),
    duration_milliseconds bigint CHECK (duration_milliseconds >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT processing_attempts_work_ownership_fk
        FOREIGN KEY (corpus_id, source_id, work_id)
        REFERENCES ingestion_work (corpus_id, source_id, id),
    UNIQUE (work_id, attempt_number),
    UNIQUE (corpus_id, source_id, id)
);

CREATE INDEX processing_attempts_source_order_idx
    ON processing_attempts (corpus_id, source_id, started_at DESC, id DESC);

CREATE TABLE source_revisions (
    id uuid PRIMARY KEY,
    source_id uuid NOT NULL,
    corpus_id uuid NOT NULL,
    attempt_id uuid NOT NULL UNIQUE,
    content_sha256 text NOT NULL CHECK (content_sha256 ~ '^[0-9a-f]{64}$'),
    captured_at timestamptz NOT NULL,
    media_type text NOT NULL CHECK (btrim(media_type) <> ''),
    byte_size bigint NOT NULL CHECK (byte_size > 0),
    pipeline_version text NOT NULL CHECK (btrim(pipeline_version) <> ''),
    final_url text CHECK (final_url IS NULL OR final_url ~ '^https://'),
    extracted_content_sha256 text CHECK (
        extracted_content_sha256 IS NULL OR extracted_content_sha256 ~ '^[0-9a-f]{64}$'
    ),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT source_revisions_attempt_ownership_fk
        FOREIGN KEY (corpus_id, source_id, attempt_id)
        REFERENCES processing_attempts (corpus_id, source_id, id),
    UNIQUE (source_id, content_sha256),
    UNIQUE (corpus_id, source_id, id)
);

CREATE TABLE document_versions (
    id uuid PRIMARY KEY,
    source_revision_id uuid NOT NULL UNIQUE,
    source_id uuid NOT NULL,
    corpus_id uuid NOT NULL,
    pipeline_version text NOT NULL CHECK (btrim(pipeline_version) <> ''),
    text_content text NOT NULL CHECK (text_content <> ''),
    text_sha256 text NOT NULL CHECK (text_sha256 ~ '^[0-9a-f]{64}$'),
    publication_status text NOT NULL DEFAULT 'published'
        CHECK (publication_status = 'published'),
    published_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT document_versions_revision_ownership_fk
        FOREIGN KEY (corpus_id, source_id, source_revision_id)
        REFERENCES source_revisions (corpus_id, source_id, id),
    UNIQUE (source_revision_id, pipeline_version, text_sha256),
    UNIQUE (corpus_id, source_id, id)
);

ALTER TABLE sources
    ADD CONSTRAINT sources_latest_document_ownership_fk
    FOREIGN KEY (corpus_id, id, latest_ready_document_id)
    REFERENCES document_versions (corpus_id, source_id, id)
    DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE document_units (
    id uuid PRIMARY KEY,
    document_id uuid NOT NULL,
    parent_id uuid,
    kind text NOT NULL CHECK (
        kind IN ('document', 'title', 'chapter', 'section', 'article', 'paragraph', 'item', 'recital', 'page', 'block')
    ),
    ordinal integer NOT NULL CHECK (ordinal >= 0),
    marker text,
    label text,
    start_offset integer NOT NULL CHECK (start_offset >= 0),
    end_offset integer NOT NULL CHECK (end_offset >= start_offset),
    start_page integer CHECK (start_page > 0),
    end_page integer CHECK (end_page >= start_page),
    locator text NOT NULL CHECK (btrim(locator) <> ''),
    content_sha256 text NOT NULL CHECK (content_sha256 ~ '^[0-9a-f]{64}$'),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT document_units_document_ownership_fk
        FOREIGN KEY (document_id) REFERENCES document_versions (id),
    UNIQUE (document_id, id),
    CONSTRAINT document_units_parent_ownership_fk
        FOREIGN KEY (document_id, parent_id) REFERENCES document_units (document_id, id),
    CONSTRAINT document_units_not_self_parent_check CHECK (parent_id IS NULL OR parent_id <> id)
);

CREATE UNIQUE INDEX document_units_locator_uidx
    ON document_units (document_id, locator);

CREATE UNIQUE INDEX document_units_parent_order_idx
    ON document_units (document_id, parent_id, ordinal) NULLS NOT DISTINCT;

INSERT INTO corpora (id, seed_key, name, description, language, jurisdiction)
VALUES
    (
        '10000000-0000-4000-8000-000000000001',
        'initial-lgpd-pt',
        'Brazilian General Data Protection Law',
        'Official Brazilian federal data-protection legislation.',
        'pt',
        'Brazil'
    ),
    (
        '10000000-0000-4000-8000-000000000002',
        'initial-gdpr-en',
        'European Union General Data Protection Regulation',
        'Official European Union data-protection regulation.',
        'en',
        'European Union'
    )
ON CONFLICT (seed_key) DO NOTHING;

INSERT INTO sources (id, corpus_id, seed_key, title, kind)
VALUES
    (
        '20000000-0000-4000-8000-000000000001',
        '10000000-0000-4000-8000-000000000001',
        'initial-lgpd-official-url',
        'Official LGPD text',
        'url'
    ),
    (
        '20000000-0000-4000-8000-000000000002',
        '10000000-0000-4000-8000-000000000002',
        'initial-gdpr-official-url',
        'Official English GDPR text',
        'url'
    )
ON CONFLICT (seed_key) DO NOTHING;

INSERT INTO url_origins (source_id, corpus_id, submitted_url, normalized_url)
VALUES
    (
        '20000000-0000-4000-8000-000000000001',
        '10000000-0000-4000-8000-000000000001',
        'https://www.planalto.gov.br/ccivil_03/_ato2015-2018/2018/lei/l13709.htm',
        'https://www.planalto.gov.br/ccivil_03/_ato2015-2018/2018/lei/l13709.htm'
    ),
    (
        '20000000-0000-4000-8000-000000000002',
        '10000000-0000-4000-8000-000000000002',
        'https://eur-lex.europa.eu/eli/reg/2016/679/oj/eng/',
        'https://eur-lex.europa.eu/eli/reg/2016/679/oj/eng/'
    )
ON CONFLICT (source_id) DO NOTHING;

INSERT INTO ingestion_work (id, source_id, corpus_id, reason)
VALUES
    (
        '30000000-0000-4000-8000-000000000001',
        '20000000-0000-4000-8000-000000000001',
        '10000000-0000-4000-8000-000000000001',
        'initial'
    ),
    (
        '30000000-0000-4000-8000-000000000002',
        '20000000-0000-4000-8000-000000000002',
        '10000000-0000-4000-8000-000000000002',
        'initial'
    )
ON CONFLICT (id) DO NOTHING;

---- create above / drop below ----

DROP TABLE IF EXISTS document_units;
ALTER TABLE sources DROP CONSTRAINT IF EXISTS sources_latest_document_ownership_fk;
DROP TABLE IF EXISTS document_versions;
DROP TABLE IF EXISTS source_revisions;
DROP TABLE IF EXISTS processing_attempts;
DROP TABLE IF EXISTS ingestion_work;
DROP TABLE IF EXISTS url_origins;
DROP TABLE IF EXISTS pdf_origins;
DROP TABLE IF EXISTS sources;
DROP TABLE IF EXISTS corpora;
