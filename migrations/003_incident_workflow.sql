ALTER TABLE incidents ADD COLUMN IF NOT EXISTS resolution_note text;
ALTER TABLE incident_activity ADD COLUMN IF NOT EXISTS note text;
