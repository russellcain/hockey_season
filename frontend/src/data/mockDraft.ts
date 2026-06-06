export type Position = 'F' | 'D' | 'G'

export interface Player {
  id: string
  name: string
  position: Position
  team: string
  salary: number
  age: number
  stats: { goals: number; assists: number; wins?: number; gaa?: number }
}

export interface DraftTeam {
  id: string
  name: string
  manager: string
  isMe: boolean
  picks: (Player | null)[]
  capUsed: number
}

export interface DraftState {
  totalRounds: number
  totalTeams: number
  currentRound: number
  currentPick: number
  status: 'in_progress'
  secondsRemaining: number
}

export const CAP_LIMIT = 82_500_000

export const SLOT_TARGETS: Record<Position, number> = { F: 9, D: 4, G: 2 }

const p = (
  id: string, name: string, pos: Position, team: string,
  salary: number, age: number, g: number, a: number,
  wins?: number, gaa?: number,
): Player => ({ id, name, position: pos, team, salary, age, stats: { goals: g, assists: a, wins, gaa } })

export const AVAILABLE_PLAYERS: Player[] = [
  p('1',  'Connor McDavid',    'F', 'EDM', 12_500_000, 28, 32, 76),
  p('2',  'Nathan MacKinnon',  'F', 'COL', 12_600_000, 29, 28, 69),
  p('3',  'Auston Matthews',   'F', 'TOR', 13_250_000, 27, 41, 45),
  p('4',  'Leon Draisaitl',    'F', 'EDM', 8_500_000,  29, 35, 57),
  p('5',  'David Pastrnak',    'F', 'BOS', 11_250_000, 28, 38, 48),
  p('6',  'Mikko Rantanen',    'F', 'CAR', 9_250_000,  27, 31, 52),
  p('7',  'Cale Makar',        'D', 'COL', 9_000_000,  26, 18, 59),
  p('8',  'Roman Josi',        'D', 'NSH', 9_059_000,  34, 12, 56),
  p('9',  'Brayden Point',     'F', 'TBL', 9_500_000,  28, 34, 52),
  p('10', 'Andrei Vasilevskiy','G', 'TBL', 9_500_000,  31,  0,  0, 37, 2.21),
  p('11', 'Igor Shesterkin',   'G', 'NYR', 11_500_000, 29,  0,  0, 41, 2.07),
  p('12', 'William Nylander',  'F', 'TOR', 11_500_000, 28, 29, 51),
  p('13', 'Sebastian Aho',     'F', 'CAR', 8_454_000,  27, 27, 49),
  p('14', 'Quinn Hughes',      'D', 'VAN', 7_850_000,  25, 10, 62),
  p('15', 'Elias Pettersson',  'F', 'VAN', 11_600_000, 26, 24, 51),
  p('16', 'Jake Guentzel',     'F', 'TBL', 6_000_000,  30, 29, 35),
  p('17', 'Viktor Arvidsson',  'F', 'LAK', 4_250_000,  31, 18, 21),
  p('18', 'Drew Doughty',      'D', 'LAK', 11_000_000, 34,  8, 31),
  p('19', 'Alex Pietrangelo',  'D', 'VGK', 8_800_000,  35,  9, 37),
  p('20', 'Kyle Connor',       'F', 'WPG', 7_142_000,  28, 28, 30),
  p('21', 'Mark Scheifele',    'F', 'WPG', 6_125_000,  32, 22, 32),
  p('22', 'Gabriel Landeskog', 'F', 'COL', 7_000_000,  32, 20, 24),
  p('23', 'Brady Tkachuk',     'F', 'OTT', 9_500_000,  25, 31, 41),
  p('24', 'Tim Stützle',       'F', 'OTT', 8_350_000,  23, 23, 47),
  p('25', 'Aleksander Barkov', 'F', 'FLA', 10_000_000, 29, 22, 53),
  p('26', 'Matthew Tkachuk',   'F', 'FLA', 9_500_000,  27, 28, 54),
  p('27', 'Sam Reinhart',      'F', 'FLA', 8_000_000,  29, 40, 30),
  p('28', 'Rasmus Dahlin',     'D', 'BUF', 8_350_000,  24, 11, 52),
  p('29', 'John Carlson',      'D', 'WSH', 8_000_000,  34,  9, 38),
  p('30', 'Nico Hischier',     'F', 'NJD', 8_250_000,  26, 26, 41),
  p('31', 'Jack Hughes',       'F', 'NJD', 8_000_000,  23, 33, 45),
  p('32', 'Frederik Andersen', 'G', 'CAR', 4_500_000,  35,  0,  0, 29, 2.45),
  p('33', 'Juuse Saros',       'G', 'NSH', 5_000_000,  29,  0,  0, 31, 2.49),
  p('34', 'Thatcher Demko',    'G', 'VAN', 5_000_000,  29,  0,  0, 28, 2.71),
  p('35', 'Boone Jenner',      'F', 'CBJ', 4_300_000,  31, 24, 22),
  p('36', 'J.T. Miller',       'F', 'VAN', 8_000_000,  31, 21, 43),
  p('37', 'Roope Hintz',       'F', 'DAL', 7_250_000,  27, 26, 38),
  p('38', 'Jason Robertson',   'F', 'DAL', 7_750_000,  25, 29, 41),
  p('39', 'Miro Heiskanen',    'D', 'DAL', 8_450_000,  25,  8, 42),
  p('40', 'Thomas Chabot',     'D', 'OTT', 8_000_000,  27,  7, 39),
  p('41', 'Linus Ullmark',    'G', 'BOS', 5_000_000,  31,  0,  0, 33, 2.38),
  p('42', 'Jake Oettinger',   'G', 'DAL', 4_750_000,  26,  0,  0, 30, 2.59),
]

const drafted = (player: Player): Player => ({ ...player })

export const TEAMS: DraftTeam[] = [
  {
    id: 't1', name: 'Frozen Flames', manager: 'Alex K.', isMe: false, capUsed: 36_000_000,
    picks: [
      drafted(AVAILABLE_PLAYERS[2]),   // Matthews
      drafted(AVAILABLE_PLAYERS[6]),   // Makar
      drafted(AVAILABLE_PLAYERS[10]),  // Shesterkin
      drafted(AVAILABLE_PLAYERS[17]),  // Doughty
      drafted(AVAILABLE_PLAYERS[20]),  // Scheifele
      null, null, null, null, null, null, null, null, null, null,
    ],
  },
  {
    id: 't2', name: 'Puck Norris', manager: 'Jamie T.', isMe: false, capUsed: 38_500_000,
    picks: [
      drafted(AVAILABLE_PLAYERS[4]),   // Pastrnak
      drafted(AVAILABLE_PLAYERS[7]),   // Josi
      drafted(AVAILABLE_PLAYERS[9]),   // Vasilevskiy
      drafted(AVAILABLE_PLAYERS[16]),  // Arvidsson
      drafted(AVAILABLE_PLAYERS[21]),  // Landeskog
      null, null, null, null, null, null, null, null, null, null,
    ],
  },
  {
    id: 't3', name: 'Hat Trick Heroes', manager: 'Sam R.', isMe: true, capUsed: 77_000_000,
    picks: [
      drafted(AVAILABLE_PLAYERS[3]),   // Draisaitl
      drafted(AVAILABLE_PLAYERS[13]),  // Q. Hughes
      drafted(AVAILABLE_PLAYERS[31]),  // Andersen     (goalie 1/2)
      drafted(AVAILABLE_PLAYERS[18]),  // Pietrangelo
      drafted(AVAILABLE_PLAYERS[26]),  // Reinhart
      drafted(AVAILABLE_PLAYERS[41]),  // Oettinger    (goalie 2/2 — slots full)
      null, null, null, null, null, null, null, null, null,
    ],
  },
  {
    id: 't4', name: 'Slapshot Squad', manager: 'Taylor M.', isMe: false, capUsed: 35_250_000,
    picks: [
      drafted(AVAILABLE_PLAYERS[5]),   // Rantanen
      drafted(AVAILABLE_PLAYERS[27]),  // Dahlin
      drafted(AVAILABLE_PLAYERS[32]),  // Saros
      drafted(AVAILABLE_PLAYERS[28]),  // Carlson
      null, null, null, null, null, null, null, null, null, null,
    ],
  },
  {
    id: 't5', name: 'Ice Cold Cash', manager: 'Morgan B.', isMe: false, capUsed: 40_000_000,
    picks: [
      drafted(AVAILABLE_PLAYERS[8]),   // Point
      drafted(AVAILABLE_PLAYERS[14]),  // Pettersson
      drafted(AVAILABLE_PLAYERS[33]),  // Demko
      drafted(AVAILABLE_PLAYERS[36]),  // Hintz
      null, null, null, null, null, null, null, null, null, null,
    ],
  },
  {
    id: 't6', name: 'Rink Rulers', manager: 'Jordan P.', isMe: false, capUsed: 33_000_000,
    picks: [
      drafted(AVAILABLE_PLAYERS[11]),  // Nylander
      drafted(AVAILABLE_PLAYERS[38]),  // Heiskanen
      drafted(AVAILABLE_PLAYERS[22]),  // Tkachuk B.
      null, null, null, null, null, null, null, null, null, null,
    ],
  },
  {
    id: 't7', name: 'Zamboni Drivers', manager: 'Casey O.', isMe: false, capUsed: 31_500_000,
    picks: [
      drafted(AVAILABLE_PLAYERS[12]),  // Aho
      drafted(AVAILABLE_PLAYERS[25]),  // Tkachuk M.
      drafted(AVAILABLE_PLAYERS[39]),  // Chabot
      null, null, null, null, null, null, null, null, null, null,
    ],
  },
  {
    id: 't8', name: 'Five Hole Fellas', manager: 'Riley W.', isMe: false, capUsed: 37_750_000,
    picks: [
      drafted(AVAILABLE_PLAYERS[1]),   // MacKinnon
      drafted(AVAILABLE_PLAYERS[29]),  // Carlson — wait, let's fix
      drafted(AVAILABLE_PLAYERS[23]),  // Stützle
      drafted(AVAILABLE_PLAYERS[35]),  // Miller
      null, null, null, null, null, null, null, null, null, null,
    ],
  },
]

export const DRAFT_STATE: DraftState = {
  totalRounds: 15,
  totalTeams: 8,
  currentRound: 3,
  currentPick: 3,
  status: 'in_progress',
  secondsRemaining: 83,
}

export function snakePickOrder(round: number, totalTeams: number): number[] {
  const order = Array.from({ length: totalTeams }, (_, i) => i)
  return round % 2 === 0 ? [...order].reverse() : order
}

export function currentPickingTeamIndex(state: DraftState): number {
  const order = snakePickOrder(state.currentRound, state.totalTeams)
  return order[state.currentPick - 1]
}

export function formatSalary(n: number): string {
  return '$' + (n / 1_000_000).toFixed(2).replace(/\.?0+$/, '') + 'M'
}
