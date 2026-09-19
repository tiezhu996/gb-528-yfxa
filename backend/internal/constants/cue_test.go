package constants

import "testing"

func TestCueTransitions(t *testing.T) {
	tests := []struct {
		name     string
		from, to CueStatus
		want     bool
	}{
		{"submit draft", CueDraft, CuePendingReview, true},
		{"review approve", CuePendingReview, CueApproved, true},
		{"review reject", CuePendingReview, CueDraft, true},
		{"lock approved", CueApproved, CueLocked, true},
		{"archive locked", CueLocked, CueArchived, true},
		{"skip review", CueDraft, CueApproved, false},
		{"unlock", CueLocked, CueApproved, false},
		{"leave archive", CueArchived, CueDraft, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := CanTransitionCue(test.from, test.to); got != test.want {
				t.Fatalf("CanTransitionCue(%q, %q) = %v, want %v", test.from, test.to, got, test.want)
			}
		})
	}
}

func TestRunTransitions(t *testing.T) {
	if !CanTransitionRun(RunEvaluated, RunPendingReview) {
		t.Fatal("evaluated run should be submittable")
	}
	if !CanTransitionRun(RunPendingReview, RunApproved) || !CanTransitionRun(RunPendingReview, RunRejected) {
		t.Fatal("pending run should accept either reviewer decision")
	}
	if CanTransitionRun(RunBlocked, RunPendingReview) {
		t.Fatal("blocked run must not be submittable")
	}
}
