ALTER TABLE fantasy_teams ADD COLUMN transactions_used INTEGER NOT NULL DEFAULT 0;
ALTER TABLE fantasy_teams ADD COLUMN trades_used INTEGER NOT NULL DEFAULT 0;
ALTER TABLE fantasy_teams ADD COLUMN league_id INTEGER REFERENCES leagues(id);
