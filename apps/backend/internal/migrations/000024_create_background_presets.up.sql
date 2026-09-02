CREATE TABLE background_presets (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    image_url TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_background_presets_public_order
    ON background_presets (sort_order DESC, id ASC)
    WHERE is_active = TRUE;

INSERT INTO background_presets (key, name, image_url, sort_order) VALUES
    ('default-01', '暮色花海', 'presets/backgrounds/default-01.webp', 50),
    ('default-02', '星夜海岸', 'presets/backgrounds/default-02.webp', 40),
    ('default-03', '青空云间', 'presets/backgrounds/default-03.webp', 30),
    ('default-04', '樱色晨光', 'presets/backgrounds/default-04.webp', 20),
    ('default-05', '静谧森林', 'presets/backgrounds/default-05.webp', 10)
ON CONFLICT (key) DO NOTHING;
