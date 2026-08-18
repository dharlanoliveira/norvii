ALTER TABLE document_versions
    DROP CONSTRAINT document_versions_source_revision_id_key;

ALTER TABLE document_versions
    ADD CONSTRAINT document_versions_revision_pipeline_unique
    UNIQUE (source_revision_id, pipeline_version);

---- create above / drop below ----

ALTER TABLE document_versions
    DROP CONSTRAINT document_versions_revision_pipeline_unique;

ALTER TABLE document_versions
    ADD CONSTRAINT document_versions_source_revision_id_key
    UNIQUE (source_revision_id);
