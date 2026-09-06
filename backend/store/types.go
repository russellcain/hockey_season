package store

type Player struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	NhlTeam     string `json:"nhlTeam"`
	NhlTeamCode string `json:"nhlTeamCode"`
	Position    string `json:"position"`
	Salary      int64  `json:"salary"`
	Age         int    `json:"age"`
	Stats       Stats  `json:"stats"`
}

type Stats struct {
	Goals   int      `json:"goals"`
	Assists int      `json:"assists"`
	Wins    *int     `json:"wins,omitempty"`
	GAA     *float64 `json:"gaa,omitempty"`
}

type Team struct {
	ID               int       `json:"id"`
	Name             string    `json:"name"`
	Manager          string    `json:"manager"`
	IsMe             bool      `json:"isMe"`
	CapUsed          int64     `json:"capUsed"`
	TransactionsUsed int       `json:"transactionsUsed"`
	TradesUsed       int       `json:"tradesUsed"`
	Picks            []*Player `json:"picks"`
}

type DraftState struct {
	ID             int    `json:"id"`
	Status         string `json:"status"`
	TotalRounds    int    `json:"totalRounds"`
	TotalTeams     int    `json:"totalTeams"`
	CurrentRound   int    `json:"currentRound"`
	CurrentPick    int    `json:"currentPick"`
	SecondsPerPick int    `json:"secondsPerPick"`
}

type DraftConfig struct {
	CapLimit    int64          `json:"capLimit"`
	SlotTargets map[string]int `json:"slotTargets"`
}

type DraftFull struct {
	DraftState DraftState  `json:"draftState"`
	Teams      []Team      `json:"teams"`
	Players    []Player    `json:"players"`
	Config     DraftConfig `json:"config"`
	MyTeamID   int         `json:"myTeamId"`
}

type PickResult struct {
	Player     Player
	DraftState DraftState
	Teams      []Team
}

// DefaultSlotTargets is the number of players per position each team must fill.
// 12 forwards, 6 defensemen, 2 goalies = 20 players total.
var DefaultSlotTargets = map[string]int{
	"F": 12,
	"D": 6,
	"G": 2,
}

// League represents a fantasy hockey league.
type League struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	SalaryCap  int64  `json:"salaryCap"`
	Status     string `json:"status"`
	SeasonYear int    `json:"seasonYear"` // year the season starts (e.g. 2025 for 2025-26)
	CreatedAt  string `json:"createdAt"`
}

// RosterSlot is a player on a team's active roster.
type RosterSlot struct {
	ID               int     `json:"id"`
	TeamID           int     `json:"teamId"`
	PlayerID         int     `json:"playerId"`
	Player           Player  `json:"player"`
	SlotType         string  `json:"slotType"` // active|injured|substitute
	OriginalPlayerID *int    `json:"originalPlayerId,omitempty"`
}

// ScoreBreakdown holds a team's scoring totals for a period.
type ScoreBreakdown struct {
	Total      float64 `json:"total"`
	Goals      int     `json:"goals"`
	Assists    int     `json:"assists"`
	GoalieWins int     `json:"goalieWins"`
	GoalieOTL  int     `json:"goalieOtl"`
	GoalieSO   int     `json:"goalieSo"`
}

// TeamStanding aggregates season-long standing data for a team.
type TeamStanding struct {
	Team        Team    `json:"team"`
	TotalPoints float64 `json:"totalPoints"`
	Goals       int     `json:"goals"`
	H2HPoints   int     `json:"h2hPoints"`
	H2HWins     int     `json:"h2hWins"`
	H2HTies     int     `json:"h2hTies"`
	H2HLosses   int     `json:"h2hLosses"`
}

// Matchup is a weekly head-to-head game between two teams.
type Matchup struct {
	ID         int     `json:"id"`
	WeekNumber int     `json:"weekNumber"`
	HomeTeamID int     `json:"homeTeamId"`
	HomeTeam   Team    `json:"homeTeam"`
	AwayTeamID int     `json:"awayTeamId"`
	AwayTeam   Team    `json:"awayTeam"`
	HomeScore  float64 `json:"homeScore"`
	AwayScore  float64 `json:"awayScore"`
	HomePoints int     `json:"homePoints"`
	AwayPoints int     `json:"awayPoints"`
}

// InjuryInfo describes an active injury flag and any substitute.
type InjuryInfo struct {
	InjuredPlayer    Player  `json:"injuredPlayer"`
	TeamID           int     `json:"teamId"`
	SubstitutePlayer *Player `json:"substitutePlayer,omitempty"`
	CapCeiling       int64   `json:"capCeiling"` // root player's cap hit
}

// TradeDetail is a trade with all legs populated.
type TradeDetail struct {
	ID              int        `json:"id"`
	Status          string     `json:"status"`
	SubmittedByTeam Team       `json:"submittedByTeam"`
	Notes           string     `json:"notes,omitempty"`
	Legs            []TradeLeg `json:"legs"`
	CreatedAt       string     `json:"createdAt"`
}

// TradeLeg is one player movement within a trade.
type TradeLeg struct {
	FromTeam Team   `json:"fromTeam"`
	ToTeam   Team   `json:"toTeam"`
	Player   Player `json:"player"`
}
