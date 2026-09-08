CREATE TABLE IF NOT EXISTS leagues (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  name        TEXT NOT NULL,
  salary_cap  INTEGER NOT NULL DEFAULT 104000000,
  status      TEXT NOT NULL DEFAULT 'setup',
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS roster_slots (
  id                 INTEGER PRIMARY KEY AUTOINCREMENT,
  team_id            INTEGER NOT NULL REFERENCES fantasy_teams(id),
  player_id          INTEGER NOT NULL REFERENCES nhl_players(id),
  league_id          INTEGER NOT NULL REFERENCES leagues(id),
  slot_type          TEXT NOT NULL DEFAULT 'active',
  original_player_id INTEGER REFERENCES nhl_players(id),
  created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(team_id, player_id, league_id)
);

CREATE TABLE IF NOT EXISTS transactions (
  id                INTEGER PRIMARY KEY AUTOINCREMENT,
  team_id           INTEGER NOT NULL REFERENCES fantasy_teams(id),
  league_id         INTEGER NOT NULL REFERENCES leagues(id),
  dropped_player_id INTEGER NOT NULL REFERENCES nhl_players(id),
  added_player_id   INTEGER NOT NULL REFERENCES nhl_players(id),
  txn_type          TEXT NOT NULL DEFAULT 'elective',
  created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS trades (
  id                   INTEGER PRIMARY KEY AUTOINCREMENT,
  league_id            INTEGER NOT NULL REFERENCES leagues(id),
  status               TEXT NOT NULL DEFAULT 'pending',
  submitted_by_team_id INTEGER NOT NULL REFERENCES fantasy_teams(id),
  reviewed_by_team_id  INTEGER REFERENCES fantasy_teams(id),
  notes                TEXT,
  created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS trade_legs (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  trade_id     INTEGER NOT NULL REFERENCES trades(id),
  from_team_id INTEGER NOT NULL REFERENCES fantasy_teams(id),
  to_team_id   INTEGER NOT NULL REFERENCES fantasy_teams(id),
  player_id    INTEGER NOT NULL REFERENCES nhl_players(id)
);

CREATE TABLE IF NOT EXISTS matchups (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  league_id    INTEGER NOT NULL REFERENCES leagues(id),
  week_number  INTEGER NOT NULL,
  home_team_id INTEGER NOT NULL REFERENCES fantasy_teams(id),
  away_team_id INTEGER NOT NULL REFERENCES fantasy_teams(id),
  home_score   REAL NOT NULL DEFAULT 0,
  away_score   REAL NOT NULL DEFAULT 0,
  home_points  INTEGER NOT NULL DEFAULT 0,
  away_points  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS player_game_logs (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  player_id INTEGER NOT NULL REFERENCES nhl_players(id),
  game_date DATE NOT NULL,
  goals     INTEGER NOT NULL DEFAULT 0,
  assists   INTEGER NOT NULL DEFAULT 0,
  wins      INTEGER NOT NULL DEFAULT 0,
  otl       INTEGER NOT NULL DEFAULT 0,
  shutouts  INTEGER NOT NULL DEFAULT 0,
  UNIQUE(player_id, game_date)
);

CREATE TABLE IF NOT EXISTS injury_flags (
  id                 INTEGER PRIMARY KEY AUTOINCREMENT,
  player_id          INTEGER NOT NULL REFERENCES nhl_players(id),
  flagged_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  resolved_at        DATETIME,
  consecutive_misses INTEGER NOT NULL DEFAULT 0,
  is_ltir            INTEGER NOT NULL DEFAULT 0
);
