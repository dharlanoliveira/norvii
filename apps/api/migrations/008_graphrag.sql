CREATE TABLE semantic_extraction_runs (
    id uuid PRIMARY KEY,
    corpus_id uuid NOT NULL REFERENCES corpora (id),
    source_id uuid NOT NULL,
    document_id uuid NOT NULL REFERENCES document_versions (id),
    extraction_version text NOT NULL CHECK (btrim(extraction_version) <> ''),
    model_identifier text NOT NULL CHECK (btrim(model_identifier) <> ''),
    input_sha256 text NOT NULL CHECK (input_sha256 ~ '^[0-9a-f]{64}$'),
    status text NOT NULL CHECK (status IN ('ready', 'failed')),
    failure_category text,
    input_tokens integer CHECK (input_tokens >= 0),
    output_tokens integer CHECK (output_tokens >= 0),
    duration_milliseconds bigint CHECK (duration_milliseconds >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    CONSTRAINT semantic_extraction_runs_document_ownership_fk
        FOREIGN KEY (corpus_id, source_id, document_id)
        REFERENCES document_versions (corpus_id, source_id, id),
    CONSTRAINT semantic_extraction_runs_status_consistency_check CHECK (
        (status = 'ready' AND failure_category IS NULL AND completed_at IS NOT NULL)
        OR (status = 'failed' AND failure_category IS NOT NULL AND completed_at IS NOT NULL)
    ),
    UNIQUE (document_id, extraction_version, input_sha256)
);

CREATE TABLE semantic_entities (
    id uuid PRIMARY KEY,
    extraction_run_id uuid NOT NULL REFERENCES semantic_extraction_runs (id),
    corpus_id uuid NOT NULL REFERENCES corpora (id),
    source_id uuid NOT NULL,
    document_id uuid NOT NULL REFERENCES document_versions (id),
    evidence_unit_id uuid NOT NULL,
    entity_type text NOT NULL CHECK (entity_type IN ('document', 'location', 'concept', 'actor', 'right', 'obligation')),
    label text NOT NULL CHECK (btrim(label) <> ''),
    normalized_label text NOT NULL CHECK (btrim(normalized_label) <> ''),
    validation_status text NOT NULL CHECK (validation_status IN ('supported', 'rejected')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT semantic_entities_document_ownership_fk
        FOREIGN KEY (corpus_id, source_id, document_id)
        REFERENCES document_versions (corpus_id, source_id, id),
    CONSTRAINT semantic_entities_evidence_ownership_fk
        FOREIGN KEY (document_id, evidence_unit_id)
        REFERENCES document_units (document_id, id),
    UNIQUE (extraction_run_id, entity_type, normalized_label, evidence_unit_id)
);

CREATE TABLE semantic_relationships (
    id uuid PRIMARY KEY,
    extraction_run_id uuid NOT NULL REFERENCES semantic_extraction_runs (id),
    corpus_id uuid NOT NULL REFERENCES corpora (id),
    source_id uuid NOT NULL,
    document_id uuid NOT NULL REFERENCES document_versions (id),
    subject_entity_id uuid NOT NULL REFERENCES semantic_entities (id),
    object_entity_id uuid NOT NULL REFERENCES semantic_entities (id),
    evidence_unit_id uuid NOT NULL,
    relationship_type text NOT NULL CHECK (relationship_type IN ('defines', 'applies_to', 'grants', 'requires', 'protects', 'governs')),
    qualifier text,
    validation_status text NOT NULL CHECK (validation_status IN ('supported', 'rejected')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT semantic_relationships_document_ownership_fk
        FOREIGN KEY (corpus_id, source_id, document_id)
        REFERENCES document_versions (corpus_id, source_id, id),
    CONSTRAINT semantic_relationships_evidence_ownership_fk
        FOREIGN KEY (document_id, evidence_unit_id)
        REFERENCES document_units (document_id, id),
    CONSTRAINT semantic_relationships_distinct_entities_check CHECK (subject_entity_id <> object_entity_id),
    UNIQUE (extraction_run_id, subject_entity_id, object_entity_id, relationship_type, evidence_unit_id)
);

CREATE TABLE graph_releases (
    id uuid PRIMARY KEY,
    corpus_id uuid NOT NULL,
    snapshot_id uuid NOT NULL,
    manifest_sha256 text NOT NULL CHECK (manifest_sha256 ~ '^[0-9a-f]{64}$'),
    build_version text NOT NULL CHECK (btrim(build_version) <> ''),
    status text NOT NULL CHECK (status IN ('building', 'ready', 'failed')),
    failure_category text,
    entity_count integer NOT NULL DEFAULT 0 CHECK (entity_count >= 0),
    relationship_count integer NOT NULL DEFAULT 0 CHECK (relationship_count >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    CONSTRAINT graph_releases_snapshot_ownership_fk
        FOREIGN KEY (snapshot_id, corpus_id)
        REFERENCES corpus_snapshots (id, corpus_id),
    CONSTRAINT graph_releases_status_consistency_check CHECK (
        (status = 'building' AND completed_at IS NULL AND failure_category IS NULL)
        OR (status = 'ready' AND completed_at IS NOT NULL AND failure_category IS NULL)
        OR (status = 'failed' AND completed_at IS NOT NULL AND failure_category IS NOT NULL)
    ),
    UNIQUE (snapshot_id, build_version, manifest_sha256)
);

CREATE TABLE graph_release_entities (
    graph_release_id uuid NOT NULL REFERENCES graph_releases (id) ON DELETE CASCADE,
    semantic_entity_id uuid NOT NULL REFERENCES semantic_entities (id),
    PRIMARY KEY (graph_release_id, semantic_entity_id)
);

CREATE TABLE graph_release_relationships (
    graph_release_id uuid NOT NULL REFERENCES graph_releases (id) ON DELETE CASCADE,
    semantic_relationship_id uuid NOT NULL REFERENCES semantic_relationships (id),
    PRIMARY KEY (graph_release_id, semantic_relationship_id)
);

CREATE INDEX semantic_extraction_runs_document_ready_idx
    ON semantic_extraction_runs (document_id, status, completed_at DESC);
CREATE INDEX semantic_entities_extraction_supported_idx
    ON semantic_entities (extraction_run_id, validation_status);
CREATE INDEX semantic_relationships_extraction_supported_idx
    ON semantic_relationships (extraction_run_id, validation_status);
CREATE INDEX graph_releases_snapshot_status_idx
    ON graph_releases (corpus_id, snapshot_id, status, completed_at DESC);

---- create above / drop below ----

DROP INDEX IF EXISTS graph_releases_snapshot_status_idx;
DROP INDEX IF EXISTS semantic_relationships_extraction_supported_idx;
DROP INDEX IF EXISTS semantic_entities_extraction_supported_idx;
DROP INDEX IF EXISTS semantic_extraction_runs_document_ready_idx;
DROP TABLE IF EXISTS graph_release_relationships;
DROP TABLE IF EXISTS graph_release_entities;
DROP TABLE IF EXISTS graph_releases;
DROP TABLE IF EXISTS semantic_relationships;
DROP TABLE IF EXISTS semantic_entities;
DROP TABLE IF EXISTS semantic_extraction_runs;
