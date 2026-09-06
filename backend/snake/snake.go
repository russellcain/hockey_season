package snake

// Order returns the team indices (0-based) to pick in the given round.
// Odd rounds go ascending [0..n-1], even rounds go descending [n-1..0].
func Order(round, totalTeams int) []int {
	order := make([]int, totalTeams)
	for i := range order {
		if round%2 == 1 {
			order[i] = i
		} else {
			order[i] = totalTeams - 1 - i
		}
	}
	return order
}

// TeamAt returns the 0-based team index picking at round/pick (both 1-based).
func TeamAt(round, pick, totalTeams int) int {
	return Order(round, totalTeams)[pick-1]
}
