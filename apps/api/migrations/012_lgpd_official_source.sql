-- Replace the unstable LGPD seed origin with the official current text hosted by the Chamber of Deputies.
UPDATE url_origins
SET submitted_url = 'https://www2.camara.leg.br/legin/fed/lei/2018/lei-13709-14-agosto-2018-787077-normaatualizada-pl.html',
    normalized_url = 'https://www2.camara.leg.br/legin/fed/lei/2018/lei-13709-14-agosto-2018-787077-normaatualizada-pl.html'
WHERE source_id = '20000000-0000-4000-8000-000000000001'
  AND corpus_id = '10000000-0000-4000-8000-000000000001';

---- create above / drop below ----

UPDATE url_origins
SET submitted_url = 'https://www.planalto.gov.br/ccivil_03/_ato2015-2018/2018/lei/l13709.htm',
    normalized_url = 'https://www.planalto.gov.br/ccivil_03/_ato2015-2018/2018/lei/l13709.htm'
WHERE source_id = '20000000-0000-4000-8000-000000000001'
  AND corpus_id = '10000000-0000-4000-8000-000000000001';
