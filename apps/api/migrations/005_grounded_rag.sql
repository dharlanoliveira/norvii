CREATE TABLE retrieval_chunks (
    id uuid PRIMARY KEY,
    corpus_id uuid NOT NULL,
    source_id uuid NOT NULL,
    document_id uuid NOT NULL,
    unit_id uuid NOT NULL,
    ordinal integer NOT NULL CHECK (ordinal >= 0),
    start_offset integer NOT NULL CHECK (start_offset >= 0),
    end_offset integer NOT NULL CHECK (end_offset > start_offset),
    content text NOT NULL CHECK (btrim(content) <> ''),
    content_sha256 text NOT NULL CHECK (content_sha256 ~ '^[0-9a-f]{64}$'),
    context_locator text NOT NULL CHECK (btrim(context_locator) <> ''),
    embedding vector(1536),
    embedding_model text,
    enrichment_status text NOT NULL DEFAULT 'pending'
        CHECK (enrichment_status IN ('pending', 'ready', 'failed')),
    enrichment_failure_category text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT retrieval_chunks_document_ownership_fk
        FOREIGN KEY (corpus_id, source_id, document_id)
        REFERENCES document_versions (corpus_id, source_id, id),
    CONSTRAINT retrieval_chunks_unit_ownership_fk
        FOREIGN KEY (document_id, unit_id)
        REFERENCES document_units (document_id, id),
    CONSTRAINT retrieval_chunks_embedding_metadata_check CHECK (
        (embedding IS NULL AND embedding_model IS NULL)
        OR (embedding IS NOT NULL AND btrim(embedding_model) <> '')
    ),
    CONSTRAINT retrieval_chunks_failure_metadata_check CHECK (
        (enrichment_status = 'failed' AND enrichment_failure_category IS NOT NULL)
        OR (enrichment_status <> 'failed' AND enrichment_failure_category IS NULL)
    ),
    UNIQUE (document_id, start_offset, end_offset),
    UNIQUE (document_id, ordinal)
);

CREATE INDEX retrieval_chunks_scope_idx
    ON retrieval_chunks (corpus_id, document_id, enrichment_status, ordinal);

CREATE INDEX retrieval_chunks_embedding_idx
    ON retrieval_chunks USING hnsw (embedding vector_cosine_ops)
    WHERE embedding IS NOT NULL;

---- create above / drop below ----

DROP INDEX IF EXISTS retrieval_chunks_embedding_idx;
DROP INDEX IF EXISTS retrieval_chunks_scope_idx;
DROP TABLE IF EXISTS retrieval_chunks;
