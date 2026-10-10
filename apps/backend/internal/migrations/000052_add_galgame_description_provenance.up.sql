-- Provenance for automatically imported descriptions. source_type keeps the
-- site's import channel; upstream_source keeps the provider's own label.
ALTER TABLE galgame_descriptions
    ADD COLUMN source_external_id VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN upstream_source VARCHAR(64) NOT NULL DEFAULT '';
