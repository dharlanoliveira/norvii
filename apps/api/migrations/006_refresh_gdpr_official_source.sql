UPDATE url_origins
SET
    submitted_url = 'https://publications.europa.eu/resource/cellar/3e485e15-11bd-11e6-ba9a-01aa75ed71a1.0006.03/DOC_1',
    normalized_url = 'https://publications.europa.eu/resource/cellar/3e485e15-11bd-11e6-ba9a-01aa75ed71a1.0006.03/DOC_1'
WHERE source_id = '20000000-0000-4000-8000-000000000002';

INSERT INTO ingestion_work (id, source_id, corpus_id, reason)
VALUES (
    '30000000-0000-4000-8000-000000000003',
    '20000000-0000-4000-8000-000000000002',
    '10000000-0000-4000-8000-000000000002',
    'reprocess'
)
ON CONFLICT (source_id) WHERE status IN ('pending', 'leased') DO NOTHING;

---- create above / drop below ----

DELETE FROM ingestion_work
WHERE id = '30000000-0000-4000-8000-000000000003';

UPDATE url_origins
SET
    submitted_url = 'https://eur-lex.europa.eu/eli/reg/2016/679/oj/eng/',
    normalized_url = 'https://eur-lex.europa.eu/eli/reg/2016/679/oj/eng/'
WHERE source_id = '20000000-0000-4000-8000-000000000002';
