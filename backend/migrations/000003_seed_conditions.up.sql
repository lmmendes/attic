-- Create default organization for initial setup
-- Use a CTE to capture the generated ID and reference it in the conditions insert
WITH org AS (
    INSERT INTO organizations (id, name, description)
    VALUES (gen_random_uuid(), 'Default Organization', 'Initial organization created during setup')
    ON CONFLICT DO NOTHING
    RETURNING id
)
INSERT INTO conditions (organization_id, code, label, description, sort_order)
VALUES
    ((SELECT id FROM org), 'NEW_SEALED', 'New (Sealed)', 'Brand new, factory sealed', 1),
    ((SELECT id FROM org), 'NEW_OPEN', 'New (Open Box)', 'New but opened, never used', 2),
    ((SELECT id FROM org), 'LIKE_NEW', 'Like New', 'Used but in excellent condition', 3),
    ((SELECT id FROM org), 'GOOD', 'Good', 'Normal wear and tear, fully functional', 4),
    ((SELECT id FROM org), 'FAIR', 'Fair', 'Some visible wear, still functional', 5),
    ((SELECT id FROM org), 'POOR', 'Poor', 'Heavy wear, may have issues', 6),
    ((SELECT id FROM org), 'FOR_PARTS', 'For Parts', 'Not functional, useful for parts only', 7)
ON CONFLICT DO NOTHING;
