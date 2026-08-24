CREATE TABLE corpus_snapshots (
    id uuid PRIMARY KEY,
    corpus_id uuid NOT NULL REFERENCES corpora (id),
    manifest_sha256 text NOT NULL CHECK (manifest_sha256 ~ '^[0-9a-f]{64}$'),
    created_by text NOT NULL CHECK (btrim(created_by) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (corpus_id, manifest_sha256),
    UNIQUE (id, corpus_id)
);

CREATE TABLE corpus_snapshot_documents (
    snapshot_id uuid NOT NULL,
    corpus_id uuid NOT NULL,
    source_id uuid NOT NULL,
    source_revision_id uuid NOT NULL,
    document_id uuid NOT NULL,
    official_origin text NOT NULL CHECK (btrim(official_origin) <> ''),
    captured_at timestamptz NOT NULL,
    content_sha256 text NOT NULL CHECK (content_sha256 ~ '^[0-9a-f]{64}$'),
    PRIMARY KEY (snapshot_id, source_id),
    CONSTRAINT corpus_snapshot_documents_snapshot_fk
        FOREIGN KEY (snapshot_id, corpus_id)
        REFERENCES corpus_snapshots (id, corpus_id),
    CONSTRAINT corpus_snapshot_documents_document_fk
        FOREIGN KEY (corpus_id, source_id, document_id)
        REFERENCES document_versions (corpus_id, source_id, id),
    CONSTRAINT corpus_snapshot_documents_revision_fk
        FOREIGN KEY (corpus_id, source_id, source_revision_id)
        REFERENCES source_revisions (corpus_id, source_id, id),
    UNIQUE (snapshot_id, document_id)
);

CREATE INDEX corpus_snapshot_documents_document_idx
    ON corpus_snapshot_documents (corpus_id, document_id, snapshot_id);

CREATE TABLE corpus_snapshot_releases (
    corpus_id uuid PRIMARY KEY REFERENCES corpora (id),
    snapshot_id uuid NOT NULL,
    version integer NOT NULL CHECK (version > 0),
    activated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT corpus_snapshot_releases_snapshot_fk
        FOREIGN KEY (snapshot_id, corpus_id)
        REFERENCES corpus_snapshots (id, corpus_id)
);

---- create above / drop below ----

DROP TABLE IF EXISTS corpus_snapshot_releases;
DROP INDEX IF EXISTS corpus_snapshot_documents_document_idx;
DROP TABLE IF EXISTS corpus_snapshot_documents;
DROP TABLE IF EXISTS corpus_snapshots;
