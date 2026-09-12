UPDATE organization_feature_settings
SET attributes = categories
WHERE attributes IS DISTINCT FROM categories;

ALTER TABLE organization_feature_settings
ADD CONSTRAINT organization_features_attributes_follow_categories
CHECK (attributes = categories);
