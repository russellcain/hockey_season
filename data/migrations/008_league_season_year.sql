-- Store the season start year on leagues so scoring queries target the right
-- calendar year regardless of when the server is running.
-- Default 2025 covers the current 2025-26 season.
ALTER TABLE leagues ADD COLUMN season_year INTEGER NOT NULL DEFAULT 2025;
