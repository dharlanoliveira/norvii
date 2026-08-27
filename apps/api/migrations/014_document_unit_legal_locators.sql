ALTER TABLE document_units
    ADD COLUMN canonical_locator text
    CHECK (
        canonical_locator IS NULL
        OR canonical_locator ~ '^[a-z][a-z-]*:[a-z0-9.-]+(/[a-z][a-z-]*:[a-z0-9.-]+)*$'
    );

CREATE UNIQUE INDEX document_units_canonical_locator_uidx
    ON document_units (document_id, canonical_locator)
    WHERE canonical_locator IS NOT NULL;

---- create above / drop below ----

DROP INDEX IF EXISTS document_units_canonical_locator_uidx;

ALTER TABLE document_units
    DROP COLUMN IF EXISTS canonical_locator;
