package constants

type CueStatus string

const (
	CueDraft         CueStatus = "draft"
	CuePendingReview CueStatus = "pending_review"
	CueApproved      CueStatus = "approved"
	CueLocked        CueStatus = "locked"
	CueArchived      CueStatus = "archived"
)

var cueTransitions = map[CueStatus]map[CueStatus]bool{
	CueDraft:         {CuePendingReview: true},
	CuePendingReview: {CueApproved: true, CueDraft: true},
	CueApproved:      {CueLocked: true, CueDraft: true},
	CueLocked:        {CueArchived: true},
	CueArchived:      {},
}

func (s CueStatus) Valid() bool {
	_, ok := cueTransitions[s]
	return ok
}

func CanTransitionCue(from, to CueStatus) bool {
	return cueTransitions[from][to]
}

type RehearsalStatus string

const (
	RunEvaluated     RehearsalStatus = "evaluated"
	RunBlocked       RehearsalStatus = "blocked"
	RunPendingReview RehearsalStatus = "pending_review"
	RunApproved      RehearsalStatus = "approved_for_rehearsal"
	RunRejected      RehearsalStatus = "rejected"
)

func CanTransitionRun(from, to RehearsalStatus) bool {
	switch from {
	case RunEvaluated:
		return to == RunPendingReview
	case RunPendingReview:
		return to == RunApproved || to == RunRejected
	default:
		return false
	}
}
