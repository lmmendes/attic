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
    ((SELECT id FROM org), 'NEW_SEALED', 'New with tags', 'A brand-new, unused item with tags attached or in the original packaging.', 1),
    ((SELECT id FROM org), 'NEW_OPEN', 'New without tags', 'A brand-new, unused item without tags or original packaging.', 2),
    ((SELECT id FROM org), 'LIKE_NEW', 'Very good', 'A lightly used item that may have slight imperfections, but still looks great.', 3),
    ((SELECT id FROM org), 'GOOD', 'Good', 'A used item that may show imperfections and signs of wear.', 4),
    ((SELECT id FROM org), 'FAIR', 'Satisfactory', 'A frequently used item with imperfections and signs of wear.', 5),
    ((SELECT id FROM org), 'POOR', 'Poor', 'Heavily used item with significant wear and visible damage. May have functional issues.', 6),
    ((SELECT id FROM org), 'FOR_PARTS', 'For parts or repair', 'Non-functional item sold as-is. Intended for parts, repair, or salvage only.', 7)
ON CONFLICT DO NOTHING;
