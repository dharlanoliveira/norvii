ALTER TABLE document_units
    DROP CONSTRAINT document_units_parent_ownership_fk;

ALTER TABLE document_units
    DROP CONSTRAINT document_units_document_id_id_key;

ALTER TABLE document_units
    DROP CONSTRAINT document_units_pkey;

ALTER TABLE document_units
    ADD CONSTRAINT document_units_pkey PRIMARY KEY (document_id, id);

ALTER TABLE document_units
    ADD CONSTRAINT document_units_parent_ownership_fk
    FOREIGN KEY (document_id, parent_id)
    REFERENCES document_units (document_id, id);

---- create above / drop below ----

ALTER TABLE document_units
    DROP CONSTRAINT document_units_parent_ownership_fk;

ALTER TABLE document_units
    DROP CONSTRAINT document_units_pkey;

ALTER TABLE document_units
    ADD CONSTRAINT document_units_pkey PRIMARY KEY (id);

ALTER TABLE document_units
    ADD CONSTRAINT document_units_document_id_id_key UNIQUE (document_id, id);

ALTER TABLE document_units
    ADD CONSTRAINT document_units_parent_ownership_fk
    FOREIGN KEY (document_id, parent_id)
    REFERENCES document_units (document_id, id);
