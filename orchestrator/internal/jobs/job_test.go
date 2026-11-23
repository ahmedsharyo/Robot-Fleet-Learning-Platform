package jobs

import "testing"

func TestTransitions(t *testing.T) {
	if !CanTransition(Queued, Leased) || !CanTransition(Running, Succeeded) || CanTransition(Succeeded, Queued) {
		t.Fatal("transition table is incorrect")
	}
}
