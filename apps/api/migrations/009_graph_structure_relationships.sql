ALTER TABLE semantic_relationships
    DROP CONSTRAINT semantic_relationships_relationship_type_check;

ALTER TABLE semantic_relationships
    ADD CONSTRAINT semantic_relationships_relationship_type_check
    CHECK (relationship_type IN (
        'contains', 'defines', 'applies_to', 'grants', 'requires', 'protects', 'governs'
    ));

---- create above / drop below ----

ALTER TABLE semantic_relationships
    DROP CONSTRAINT semantic_relationships_relationship_type_check;

ALTER TABLE semantic_relationships
    ADD CONSTRAINT semantic_relationships_relationship_type_check
    CHECK (relationship_type IN (
        'defines', 'applies_to', 'grants', 'requires', 'protects', 'governs'
    ));
