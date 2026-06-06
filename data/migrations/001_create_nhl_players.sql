-- Migration 001: NHL player reference table
-- Read-only reference table seeded once per season via draft/seed.
-- Idempotent: safe to re-run; INSERT OR IGNORE on (name, nhl_team_code) prevents duplicates.

CREATE TABLE IF NOT EXISTS nhl_players (
    id             INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    nhl_id         INTEGER,                   -- null when name-matching failed; patch manually
    name           TEXT    NOT NULL,
    nhl_team_name  TEXT    NOT NULL,
    nhl_team_code  TEXT    NOT NULL,
    position       TEXT    NOT NULL,          -- F, D, or G
    salary_cap_hit TEXT    NOT NULL,
    age            TEXT    NOT NULL,
    UNIQUE(name, nhl_team_code)
);
