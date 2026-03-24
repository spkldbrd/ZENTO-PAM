INSERT INTO policies (name, rules)
SELECT 'default', '{"allowed_publishers": ["Microsoft Corporation"]}'::jsonb
WHERE NOT EXISTS (SELECT 1 FROM policies WHERE name = 'default');
