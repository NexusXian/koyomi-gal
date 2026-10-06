CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE companies (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    name_ja VARCHAR(255) NOT NULL DEFAULT '',
    name_zh VARCHAR(255) NOT NULL DEFAULT '',
    name_en VARCHAR(255) NOT NULL DEFAULT '',
    slug VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    logo_url TEXT NOT NULL DEFAULT '',
    banner_url TEXT NOT NULL DEFAULT '',
    website_url TEXT NOT NULL DEFAULT '',
    country VARCHAR(100) NOT NULL DEFAULT '',
    founded_date DATE,
    closed_date DATE,
    status VARCHAR(16) NOT NULL CHECK (status IN ('active', 'inactive', 'closed')),
    parent_company_id BIGINT REFERENCES companies(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT companies_not_self_parent CHECK (parent_company_id <> id),
    CONSTRAINT companies_date_order CHECK (closed_date >= founded_date)
);
CREATE UNIQUE INDEX companies_slug_unique ON companies(slug);
CREATE INDEX idx_companies_parent ON companies(parent_company_id);
CREATE INDEX idx_companies_status_country ON companies(status, country);
CREATE INDEX idx_companies_name ON companies(name);
CREATE INDEX idx_companies_name_trgm ON companies USING GIN (name gin_trgm_ops);
CREATE INDEX idx_companies_name_ja_trgm ON companies USING GIN (name_ja gin_trgm_ops);
CREATE INDEX idx_companies_name_zh_trgm ON companies USING GIN (name_zh gin_trgm_ops);
CREATE INDEX idx_companies_name_en_trgm ON companies USING GIN (name_en gin_trgm_ops);
CREATE INDEX idx_companies_created_at ON companies(created_at DESC, id DESC);

CREATE TABLE company_aliases (
    id BIGSERIAL PRIMARY KEY,
    company_id BIGINT NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    alias VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT company_aliases_unique UNIQUE (company_id, alias)
);
CREATE INDEX idx_company_aliases_alias_trgm ON company_aliases USING GIN (alias gin_trgm_ops);

CREATE TABLE game_companies (
    game_id BIGINT NOT NULL REFERENCES galgames(id) ON DELETE CASCADE,
    company_id BIGINT NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    role VARCHAR(16) NOT NULL CHECK (role IN ('developer', 'publisher', 'brand', 'co_developer')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (game_id, company_id, role)
);
CREATE INDEX idx_game_companies_company_role ON game_companies(company_id, role, game_id);

INSERT INTO companies (id, name, name_ja, slug, description, logo_url, website_url, status, created_at, updated_at)
SELECT id, name, original_name, slug, description, logo_url, website, 'active', created_at, updated_at
FROM developers;
SELECT setval(pg_get_serial_sequence('companies', 'id'), COALESCE(MAX(id), 1), MAX(id) IS NOT NULL) FROM companies;
INSERT INTO company_aliases (company_id, alias)
SELECT id, BTRIM(original_name) FROM developers WHERE BTRIM(original_name) <> '';
INSERT INTO game_companies (game_id, company_id, role)
SELECT id, developer_id, 'developer' FROM galgames WHERE developer_id IS NOT NULL;

INSERT INTO permissions (name, code, resource, action, description)
VALUES ('Manage companies', 'company:manage', 'company', 'manage', 'Manage companies and game company associations')
ON CONFLICT (code) DO NOTHING;
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.code IN ('admin', 'super_admin') AND p.code = 'company:manage'
ON CONFLICT DO NOTHING;
