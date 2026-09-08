package snake_test

import (
	"testing"

	"hockey_season/backend/snake"
)

func TestOrderOddRound(t *testing.T) {
	got := snake.Order(1, 4)
	want := []int{0, 1, 2, 3}
	for i, v := range got {
		if v != want[i] {
			t.Fatalf("round 1 order[%d]: got %d, want %d", i, v, want[i])
		}
	}
}

func TestOrderEvenRound(t *testing.T) {
	got := snake.Order(2, 4)
	want := []int{3, 2, 1, 0}
	for i, v := range got {
		if v != want[i] {
			t.Fatalf("round 2 order[%d]: got %d, want %d", i, v, want[i])
		}
	}
}

func TestOrderRound3IsSameAsRound1(t *testing.T) {
	got := snake.Order(3, 4)
	want := snake.Order(1, 4)
	for i, v := range got {
		if v != want[i] {
			t.Fatalf("round 3 order[%d]: got %d, want %d", i, v, want[i])
		}
	}
}

func TestTeamAt(t *testing.T) {
	cases := []struct {
		round, pick, total int
		want               int
	}{
		{1, 1, 8, 0}, // first pick of round 1 → team 0
		{1, 8, 8, 7}, // last pick of round 1 → team 7
		{2, 1, 8, 7}, // first pick of round 2 → team 7 (reversed)
		{2, 8, 8, 0}, // last pick of round 2 → team 0
		{3, 1, 8, 0}, // round 3 repeats round 1
	}
	for _, tc := range cases {
		got := snake.TeamAt(tc.round, tc.pick, tc.total)
		if got != tc.want {
			t.Errorf("TeamAt(%d,%d,%d) = %d, want %d", tc.round, tc.pick, tc.total, got, tc.want)
		}
	}
}
