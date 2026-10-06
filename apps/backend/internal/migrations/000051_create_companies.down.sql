DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'company:manage');
DELETE FROM permissions WHERE code = 'company:manage';
DROP TABLE game_companies;
DROP TABLE company_aliases;
DROP TABLE companies;
