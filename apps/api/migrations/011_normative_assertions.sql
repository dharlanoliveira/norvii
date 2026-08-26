DROP TABLE graph_release_relationships;
DROP TABLE semantic_relationships;

ALTER TABLE semantic_entities
    DROP CONSTRAINT semantic_entities_entity_type_check;

ALTER TABLE semantic_entities
    ADD CONSTRAINT semantic_entities_entity_type_check
    CHECK (entity_type IN (
        'document', 'location', 'concept', 'actor', 'right', 'obligation', 'condition'
    ));

-- Legacy structural entities remain readable until the reset. This constraint is
-- deliberately NOT VALID so it governs all new writes without rejecting old rows.
ALTER TABLE semantic_entities
    ADD CONSTRAINT semantic_entities_legal_entity_type_check
    CHECK (entity_type IN ('concept', 'actor', 'right', 'obligation', 'condition')) NOT VALID;

ALTER TABLE semantic_entities
    ADD CONSTRAINT semantic_entities_document_id_id_key UNIQUE (document_id, id);

CREATE TABLE normative_assertions (
    id uuid PRIMARY KEY,
    extraction_run_id uuid NOT NULL REFERENCES semantic_extraction_runs (id),
    corpus_id uuid NOT NULL REFERENCES corpora (id),
    source_id uuid NOT NULL,
    document_id uuid NOT NULL REFERENCES document_versions (id),
    subject_entity_id uuid NOT NULL,
    object_entity_id uuid NOT NULL,
    establishing_unit_id uuid NOT NULL,
    evidence_unit_id uuid NOT NULL,
    predicate text NOT NULL CHECK (predicate IN (
        'defines', 'applies_to', 'must_be_observed_by', 'imposes_duty_on',
        'grants', 'protects', 'assigns_responsibility_to', 'conditions'
    )),
    qualifier text,
    validation_status text NOT NULL CHECK (validation_status IN ('supported', 'rejected')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT normative_assertions_document_ownership_fk
        FOREIGN KEY (corpus_id, source_id, document_id)
        REFERENCES document_versions (corpus_id, source_id, id),
    CONSTRAINT normative_assertions_subject_entity_fk
        FOREIGN KEY (document_id, subject_entity_id)
        REFERENCES semantic_entities (document_id, id),
    CONSTRAINT normative_assertions_object_entity_fk
        FOREIGN KEY (document_id, object_entity_id)
        REFERENCES semantic_entities (document_id, id),
    CONSTRAINT normative_assertions_establishing_unit_fk
        FOREIGN KEY (document_id, establishing_unit_id)
        REFERENCES document_units (document_id, id),
    CONSTRAINT normative_assertions_evidence_unit_fk
        FOREIGN KEY (document_id, evidence_unit_id)
        REFERENCES document_units (document_id, id),
    CONSTRAINT normative_assertions_distinct_entities_check
        CHECK (subject_entity_id <> object_entity_id),
    CONSTRAINT normative_assertions_qualifier_check
        CHECK (qualifier IS NULL OR btrim(qualifier) <> ''),
    UNIQUE (
        extraction_run_id, subject_entity_id, object_entity_id, predicate,
        establishing_unit_id, evidence_unit_id
    )
);

CREATE TABLE graph_release_legal_units (
    graph_release_id uuid NOT NULL REFERENCES graph_releases (id) ON DELETE CASCADE,
    document_id uuid NOT NULL,
    legal_unit_id uuid NOT NULL,
    PRIMARY KEY (graph_release_id, document_id, legal_unit_id),
    FOREIGN KEY (document_id, legal_unit_id)
        REFERENCES document_units (document_id, id)
);

CREATE TABLE graph_release_assertions (
    graph_release_id uuid NOT NULL REFERENCES graph_releases (id) ON DELETE CASCADE,
    normative_assertion_id uuid NOT NULL REFERENCES normative_assertions (id),
    PRIMARY KEY (graph_release_id, normative_assertion_id)
);

CREATE INDEX normative_assertions_extraction_supported_idx
    ON normative_assertions (extraction_run_id, validation_status);

---- create above / drop below ----

DROP INDEX IF EXISTS normative_assertions_extraction_supported_idx;
DROP TABLE IF EXISTS graph_release_assertions;
DROP TABLE IF EXISTS graph_release_legal_units;
DROP TABLE IF EXISTS normative_assertions;

ALTER TABLE semantic_entities
    DROP CONSTRAINT IF EXISTS semantic_entities_document_id_id_key;

ALTER TABLE semantic_entities
    DROP CONSTRAINT IF EXISTS semantic_entities_legal_entity_type_check;

CREATE TABLE semantic_relationships (
    id uuid PRIMARY KEY,
    extraction_run_id uuid NOT NULL REFERENCES semantic_extraction_runs (id),
    corpus_id uuid NOT NULL REFERENCES corpora (id),
    source_id uuid NOT NULL,
    document_id uuid NOT NULL REFERENCES document_versions (id),
    subject_entity_id uuid NOT NULL REFERENCES semantic_entities (id),
    object_entity_id uuid NOT NULL REFERENCES semantic_entities (id),
    evidence_unit_id uuid NOT NULL,
    relationship_type text NOT NULL CHECK (relationship_type IN (
        'contains', 'defines', 'applies_to', 'grants', 'requires', 'protects', 'governs'
    )),
    qualifier text,
    validation_status text NOT NULL CHECK (validation_status IN ('supported', 'rejected')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT semantic_relationships_document_ownership_fk
        FOREIGN KEY (corpus_id, source_id, document_id)
        REFERENCES document_versions (corpus_id, source_id, id),
    CONSTRAINT semantic_relationships_evidence_ownership_fk
        FOREIGN KEY (document_id, evidence_unit_id)
        REFERENCES document_units (document_id, id),
    CONSTRAINT semantic_relationships_distinct_entities_check
        CHECK (subject_entity_id <> object_entity_id),
    UNIQUE (extraction_run_id, subject_entity_id, object_entity_id, relationship_type, evidence_unit_id)
);

CREATE TABLE graph_release_relationships (
    graph_release_id uuid NOT NULL REFERENCES graph_releases (id) ON DELETE CASCADE,
    semantic_relationship_id uuid NOT NULL REFERENCES semantic_relationships (id),
    PRIMARY KEY (graph_release_id, semantic_relationship_id)
);

CREATE INDEX semantic_relationships_extraction_supported_idx
    ON semantic_relationships (extraction_run_id, validation_status);
