INSERT INTO categories (organization_id, name, description)
SELECT DISTINCT a.organization_id, 'Uncategorized', 'Fallback category created while rolling back optional asset categories.'
FROM assets a
WHERE a.category_id IS NULL
  AND NOT EXISTS (
    SELECT 1
    FROM categories c
    WHERE c.organization_id = a.organization_id
      AND c.name = 'Uncategorized'
  );

UPDATE assets a
SET category_id = (
  SELECT c.id
  FROM categories c
  WHERE c.organization_id = a.organization_id
    AND c.name = 'Uncategorized'
  ORDER BY c.created_at
  LIMIT 1
)
WHERE a.category_id IS NULL;

ALTER TABLE assets ALTER COLUMN category_id SET NOT NULL;
