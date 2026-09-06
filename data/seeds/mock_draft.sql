-- Mock draft seed — mirrors the state in frontend/src/data/mockDraft.ts.
-- Intended for development with DRAFT_SECRET=dev-secret.
-- Executed programmatically by backend/mockdata/seed.go (not by the migration runner).
-- Player lookups use name + nhl_team_code subqueries so IDs stay portable.
-- cap_used is set directly from mock values, not computed from picks.

-- ── Fantasy Teams ───────────────────────────────────────────────────────────
-- code_hash = HMAC-SHA256(code, DRAFT_SECRET).  Values below assume dev-secret;
-- the Go seeder computes them at runtime so this file always reflects intent.
INSERT OR IGNORE INTO fantasy_teams (name, manager, code_hash, cap_used) VALUES
  ('Frozen Flames',    'Alex K.',    '<computed: frozen-flames-code>',    36000000),
  ('Puck Norris',      'Jamie T.',   '<computed: puck-norris-code>',      38500000),
  ('Hat Trick Heroes', 'Sam R.',     '<computed: hat-trick-heroes-code>', 77000000),
  ('Slapshot Squad',   'Taylor M.',  '<computed: slapshot-squad-code>',   35250000),
  ('Ice Cold Cash',    'Morgan B.',  '<computed: ice-cold-cash-code>',    40000000),
  ('Rink Rulers',      'Jordan P.',  '<computed: rink-rulers-code>',      33000000),
  ('Zamboni Drivers',  'Casey O.',   '<computed: zamboni-drivers-code>',  31500000),
  ('Five Hole Fellas', 'Riley W.',   '<computed: five-hole-fellas-code>', 37750000);

-- ── Draft Session ────────────────────────────────────────────────────────────
-- current state: round 3, pick 3 → Hat Trick Heroes (team index 2) is up next.
INSERT OR IGNORE INTO draft_sessions
  (status, total_rounds, total_teams, current_round, current_pick, seconds_per_pick, cap_limit)
VALUES
  ('in_progress', 15, 8, 3, 3, 90, 82500000);

-- ── Draft Picks ──────────────────────────────────────────────────────────────
-- Snake order:  odd rounds  → team 0..7 picks 1..8
--               even rounds → team 7..0 picks 1..8
--
-- Round 1 (ascending)
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 1, 1 FROM fantasy_teams t, nhl_players p WHERE t.name='Frozen Flames'    AND p.name='Auston Matthews';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 1, 2 FROM fantasy_teams t, nhl_players p WHERE t.name='Puck Norris'      AND p.name='David Pastrnak';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 1, 3 FROM fantasy_teams t, nhl_players p WHERE t.name='Hat Trick Heroes' AND p.name='Leon Draisaitl';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 1, 4 FROM fantasy_teams t, nhl_players p WHERE t.name='Slapshot Squad'   AND p.name='Mikko Rantanen';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 1, 5 FROM fantasy_teams t, nhl_players p WHERE t.name='Ice Cold Cash'    AND p.name='Brayden Point';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 1, 6 FROM fantasy_teams t, nhl_players p WHERE t.name='Rink Rulers'      AND p.name='William Nylander';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 1, 7 FROM fantasy_teams t, nhl_players p WHERE t.name='Zamboni Drivers'  AND p.name='Sebastian Aho';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 1, 8 FROM fantasy_teams t, nhl_players p WHERE t.name='Five Hole Fellas' AND p.name='Nathan MacKinnon';

-- Round 2 (descending)
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 2, 1 FROM fantasy_teams t, nhl_players p WHERE t.name='Five Hole Fellas' AND p.name='Nico Hischier';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 2, 2 FROM fantasy_teams t, nhl_players p WHERE t.name='Zamboni Drivers'  AND p.name='Matthew Tkachuk';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 2, 3 FROM fantasy_teams t, nhl_players p WHERE t.name='Rink Rulers'      AND p.name='Miro Heiskanen';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 2, 4 FROM fantasy_teams t, nhl_players p WHERE t.name='Ice Cold Cash'    AND p.name='Elias Pettersson';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 2, 5 FROM fantasy_teams t, nhl_players p WHERE t.name='Slapshot Squad'   AND p.name='Rasmus Dahlin';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 2, 6 FROM fantasy_teams t, nhl_players p WHERE t.name='Hat Trick Heroes' AND p.name='Quinn Hughes';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 2, 7 FROM fantasy_teams t, nhl_players p WHERE t.name='Puck Norris'      AND p.name='Roman Josi';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 2, 8 FROM fantasy_teams t, nhl_players p WHERE t.name='Frozen Flames'    AND p.name='Cale Makar';

-- Round 3 (ascending) — all 8 picks pre-seeded; current state = pick 3 next
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 3, 1 FROM fantasy_teams t, nhl_players p WHERE t.name='Frozen Flames'    AND p.name='Igor Shesterkin';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 3, 2 FROM fantasy_teams t, nhl_players p WHERE t.name='Puck Norris'      AND p.name='Andrei Vasilevskiy';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 3, 3 FROM fantasy_teams t, nhl_players p WHERE t.name='Hat Trick Heroes' AND p.name='Frederik Andersen';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 3, 4 FROM fantasy_teams t, nhl_players p WHERE t.name='Slapshot Squad'   AND p.name='Juuse Saros';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 3, 5 FROM fantasy_teams t, nhl_players p WHERE t.name='Ice Cold Cash'    AND p.name='Thatcher Demko';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 3, 6 FROM fantasy_teams t, nhl_players p WHERE t.name='Rink Rulers'      AND p.name='Brady Tkachuk';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 3, 7 FROM fantasy_teams t, nhl_players p WHERE t.name='Zamboni Drivers'  AND p.name='Thomas Chabot';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 3, 8 FROM fantasy_teams t, nhl_players p WHERE t.name='Five Hole Fellas' AND p.name='Tim Stutzle';

-- Round 4 (descending)
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 4, 1 FROM fantasy_teams t, nhl_players p WHERE t.name='Five Hole Fellas' AND p.name='J.T. Miller';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 4, 4 FROM fantasy_teams t, nhl_players p WHERE t.name='Ice Cold Cash'    AND p.name='Roope Hintz';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 4, 5 FROM fantasy_teams t, nhl_players p WHERE t.name='Slapshot Squad'   AND p.name='John Carlson';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 4, 6 FROM fantasy_teams t, nhl_players p WHERE t.name='Hat Trick Heroes' AND p.name='Alex Pietrangelo';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 4, 7 FROM fantasy_teams t, nhl_players p WHERE t.name='Puck Norris'      AND p.name='Viktor Arvidsson';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 4, 8 FROM fantasy_teams t, nhl_players p WHERE t.name='Frozen Flames'    AND p.name='Drew Doughty';

-- Round 5 (ascending)
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 5, 1 FROM fantasy_teams t, nhl_players p WHERE t.name='Frozen Flames'    AND p.name='Mark Scheifele';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 5, 2 FROM fantasy_teams t, nhl_players p WHERE t.name='Puck Norris'      AND p.name='Gabriel Landeskog';
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 5, 3 FROM fantasy_teams t, nhl_players p WHERE t.name='Hat Trick Heroes' AND p.name='Sam Reinhart';

-- Round 6 (descending) — Hat Trick Heroes at pick 6 fills their 2nd goalie slot
INSERT OR IGNORE INTO draft_picks (session_id, team_id, player_id, round, pick_number)
SELECT 1, t.id, p.id, 6, 6 FROM fantasy_teams t, nhl_players p WHERE t.name='Hat Trick Heroes' AND p.name='Jake Oettinger';
