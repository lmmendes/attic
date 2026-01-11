-- Create default organization for initial setup
INSERT INTO organizations (id, name, description)
VALUES ('00000000-0000-0000-0000-000000000001', 'Default Organization', 'Initial organization created during setup')
ON CONFLICT DO NOTHING;

-- Seed default conditions
INSERT INTO conditions (organization_id, code, label, description, sort_order)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'NEW_SEALED', 'New (Sealed)', 'Brand new, factory sealed', 1),
    ('00000000-0000-0000-0000-000000000001', 'NEW_OPEN', 'New (Open Box)', 'New but opened, never used', 2),
    ('00000000-0000-0000-0000-000000000001', 'LIKE_NEW', 'Like New', 'Used but in excellent condition', 3),
    ('00000000-0000-0000-0000-000000000001', 'GOOD', 'Good', 'Normal wear and tear, fully functional', 4),
    ('00000000-0000-0000-0000-000000000001', 'FAIR', 'Fair', 'Some visible wear, still functional', 5),
    ('00000000-0000-0000-0000-000000000001', 'POOR', 'Poor', 'Heavy wear, may have issues', 6),
    ('00000000-0000-0000-0000-000000000001', 'FOR_PARTS', 'For Parts', 'Not functional, useful for parts only', 7)
ON CONFLICT DO NOTHING;
