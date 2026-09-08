CREATE TABLE IF NOT EXISTS draft_sessions (
  id               INTEGER PRIMARY KEY AUTOINCREMENT,
  status           TEXT NOT NULL DEFAULT 'waiting',
  total_rounds     INTEGER NOT NULL DEFAULT 15,
  total_teams      INTEGER NOT NULL DEFAULT 8,
  current_round    INTEGER NOT NULL DEFAULT 1,
  current_pick     INTEGER NOT NULL DEFAULT 1,
  seconds_per_pick INTEGER NOT NULL DEFAULT 90,
  cap_limit        INTEGER NOT NULL DEFAULT 82500000
);

CREATE TABLE IF NOT EXISTS draft_picks (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id  INTEGER NOT NULL REFERENCES draft_sessions(id),
  team_id     INTEGER NOT NULL REFERENCES fantasy_teams(id),
  player_id   INTEGER NOT NULL REFERENCES nhl_players(id),
  round       INTEGER NOT NULL,
  pick_number INTEGER NOT NULL,
  drafted_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (session_id, player_id)
);
