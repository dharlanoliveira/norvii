-- Refresh unstable U.S. fair-housing origins with durable government-hosted PDFs.
UPDATE url_origins
SET submitted_url = 'https://www.govinfo.gov/content/pkg/USCODE-2024-title42/pdf/USCODE-2024-title42-chap45-subchapI-sec3604.pdf',
    normalized_url = 'https://www.govinfo.gov/content/pkg/USCODE-2024-title42/pdf/USCODE-2024-title42-chap45-subchapI-sec3604.pdf'
WHERE source_id = '20000000-0000-4000-8000-000000000008'
  AND corpus_id = '10000000-0000-4000-8000-000000000004';

UPDATE url_origins
SET submitted_url = 'https://www.govinfo.gov/content/pkg/CFR-2025-title24-vol1/pdf/CFR-2025-title24-vol1-sec103-25.pdf',
    normalized_url = 'https://www.govinfo.gov/content/pkg/CFR-2025-title24-vol1/pdf/CFR-2025-title24-vol1-sec103-25.pdf'
WHERE source_id = '20000000-0000-4000-8000-000000000011'
  AND corpus_id = '10000000-0000-4000-8000-000000000004';

-- Reprocess only when the source has no work currently pending or leased.
INSERT INTO ingestion_work (id, source_id, corpus_id, reason)
SELECT queue.id, queue.source_id, queue.corpus_id, 'reprocess'
FROM (
    VALUES
        (
            '30000000-0000-4000-8000-000000000014'::uuid,
            '20000000-0000-4000-8000-000000000008'::uuid,
            '10000000-0000-4000-8000-000000000004'::uuid
        ),
        (
            '30000000-0000-4000-8000-000000000015'::uuid,
            '20000000-0000-4000-8000-000000000010'::uuid,
            '10000000-0000-4000-8000-000000000004'::uuid
        ),
        (
            '30000000-0000-4000-8000-000000000016'::uuid,
            '20000000-0000-4000-8000-000000000011'::uuid,
            '10000000-0000-4000-8000-000000000004'::uuid
        )
) AS queue(id, source_id, corpus_id)
JOIN sources
    ON sources.id = queue.source_id
   AND sources.corpus_id = queue.corpus_id
WHERE NOT EXISTS (
    SELECT 1
    FROM ingestion_work AS existing_work
    WHERE existing_work.source_id = queue.source_id
      AND existing_work.corpus_id = queue.corpus_id
      AND existing_work.status IN ('pending', 'leased')
)
ON CONFLICT DO NOTHING;

---- create above / drop below ----

DELETE FROM ingestion_work
WHERE id IN (
    '30000000-0000-4000-8000-000000000014',
    '30000000-0000-4000-8000-000000000015',
    '30000000-0000-4000-8000-000000000016'
)
AND reason = 'reprocess';

UPDATE url_origins
SET submitted_url = 'https://uscode.house.gov/view.xhtml?req=%28title%3A42+section%3A3604+edition%3Aprelim%29',
    normalized_url = 'https://uscode.house.gov/view.xhtml?req=%28title%3A42+section%3A3604+edition%3Aprelim%29'
WHERE source_id = '20000000-0000-4000-8000-000000000008'
  AND corpus_id = '10000000-0000-4000-8000-000000000004'
  AND normalized_url = 'https://www.govinfo.gov/content/pkg/USCODE-2024-title42/pdf/USCODE-2024-title42-chap45-subchapI-sec3604.pdf';

UPDATE url_origins
SET submitted_url = 'https://www.hud.gov/fairhousing/fileacomplaint',
    normalized_url = 'https://www.hud.gov/fairhousing/fileacomplaint'
WHERE source_id = '20000000-0000-4000-8000-000000000011'
  AND corpus_id = '10000000-0000-4000-8000-000000000004'
  AND normalized_url = 'https://www.govinfo.gov/content/pkg/CFR-2025-title24-vol1/pdf/CFR-2025-title24-vol1-sec103-25.pdf';
