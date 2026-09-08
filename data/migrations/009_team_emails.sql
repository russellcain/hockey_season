-- Migration 009: per-team email address for notifications
ALTER TABLE fantasy_teams ADD COLUMN email TEXT;
