CREATE TABLE galgame_descriptions (
    id BIGSERIAL PRIMARY KEY,
    galgame_id BIGINT NOT NULL,
    language VARCHAR(16) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    source_type VARCHAR(32) NOT NULL DEFAULT 'unknown',
    source_name VARCHAR(128) NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    is_official BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_galgame_descriptions_galgame FOREIGN KEY (galgame_id)
        REFERENCES galgames (id) ON DELETE CASCADE,
    CONSTRAINT galgame_descriptions_language_check
        CHECK (language IN ('zh-CN', 'en-US', 'ja-JP')),
    CONSTRAINT galgame_descriptions_source_type_check
        CHECK (source_type IN ('unknown', 'vndb', 'bangumi', 'nextmoe', 'official', 'steam', 'manual')),
    CONSTRAINT galgame_descriptions_galgame_id_language_key UNIQUE (galgame_id, language)
);

CREATE INDEX galgame_descriptions_galgame_id_idx ON galgame_descriptions (galgame_id);

-- The legacy column now mirrors the zh-CN row's source, which may use the
-- expanded per-language source set.
ALTER TABLE galgames DROP CONSTRAINT IF EXISTS galgames_description_source_check;
ALTER TABLE galgames
    ADD CONSTRAINT galgames_description_source_check
        CHECK (description_source IN ('', 'unknown', 'vndb', 'bangumi', 'nextmoe', 'official', 'steam', 'manual'));

-- Migrate the legacy single description column into per-language rows.
-- 'vndb' descriptions are English text, everything else is treated as
-- Chinese; unknown-provenance rows become manual/旧数据 instead of
-- guessing a provider.
INSERT INTO galgame_descriptions
    (galgame_id, language, content, source_type, source_name, source_url, is_official)
SELECT g.id,
       'en-US',
       g.description,
       'vndb',
       'VNDB',
       COALESCE((
           SELECT es.url FROM galgame_external_sources es
           WHERE es.galgame_id = g.id AND es.source = 'vndb'
           ORDER BY es.id LIMIT 1
       ), ''),
       FALSE
FROM galgames g
WHERE g.description <> '' AND g.description_source = 'vndb'
ON CONFLICT DO NOTHING;

INSERT INTO galgame_descriptions
    (galgame_id, language, content, source_type, source_name, source_url, is_official)
SELECT g.id,
       'zh-CN',
       g.description,
       'bangumi',
       'Bangumi',
       COALESCE((
           SELECT es.url FROM galgame_external_sources es
           WHERE es.galgame_id = g.id AND es.source = 'bangumi'
           ORDER BY es.id LIMIT 1
       ), ''),
       FALSE
FROM galgames g
WHERE g.description <> '' AND g.description_source = 'bangumi'
ON CONFLICT DO NOTHING;

INSERT INTO galgame_descriptions
    (galgame_id, language, content, source_type, source_name, source_url, is_official)
SELECT g.id,
       'zh-CN',
       g.description,
       'manual',
       CASE WHEN g.description_source = 'manual' THEN '手动录入' ELSE '旧数据' END,
       '',
       FALSE
FROM galgames g
WHERE g.description <> ''
  AND g.description_source NOT IN ('vndb', 'bangumi')
ON CONFLICT DO NOTHING;
