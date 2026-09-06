-- Link draft sessions to leagues so FinaliseDraft knows which league to populate.
ALTER TABLE draft_sessions ADD COLUMN league_id INTEGER REFERENCES leagues(id);
