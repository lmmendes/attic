DELETE FROM conditions WHERE organization_id = (SELECT id FROM organizations WHERE name = 'Default Organization');
DELETE FROM organizations WHERE name = 'Default Organization';
