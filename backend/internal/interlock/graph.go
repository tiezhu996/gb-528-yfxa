package interlock

import (
	"fmt"
	"sort"
	"strings"
)

type GraphCue struct {
	ID            uint   `json:"id"`
	CueCode       string `json:"cue_code"`
	SequenceNo    int    `json:"sequence_no"`
	StartOffsetMS int64  `json:"start_offset_ms"`
	DurationMS    int64  `json:"duration_ms"`
	DependencyIDs []uint `json:"dependency_ids"`
}

type GraphError struct {
	Code                string   `json:"code"`
	Message             string   `json:"message"`
	EvidencePath        []string `json:"evidence_path,omitempty"`
	MissingDependencyID uint     `json:"missing_dependency_id,omitempty"`
	DuplicateSequence   int      `json:"duplicate_sequence,omitempty"`
}

func (e *GraphError) Error() string {
	if len(e.EvidencePath) > 0 {
		return e.Message + ": " + strings.Join(e.EvidencePath, " -> ")
	}
	return e.Message
}

func ValidateAndSort(cues []GraphCue) ([]GraphCue, error) {
	if len(cues) == 0 {
		return nil, &GraphError{Code: "EMPTY_CUE_SET", Message: "at least one locked cue is required"}
	}
	byID := make(map[uint]GraphCue, len(cues))
	sequenceOwner := make(map[int]string, len(cues))
	for _, cue := range cues {
		if cue.ID == 0 || strings.TrimSpace(cue.CueCode) == "" {
			return nil, &GraphError{Code: "INVALID_CUE", Message: "every cue must have an id and code"}
		}
		if cue.DurationMS <= 0 || cue.StartOffsetMS < 0 {
			return nil, &GraphError{Code: "INVALID_CUE_TIMING", Message: fmt.Sprintf("cue %s has invalid timing", cue.CueCode), EvidencePath: []string{cue.CueCode}}
		}
		if owner, exists := sequenceOwner[cue.SequenceNo]; exists {
			return nil, &GraphError{Code: "DUPLICATE_CUE_SEQUENCE", Message: fmt.Sprintf("sequence %d is shared by %s and %s", cue.SequenceNo, owner, cue.CueCode), EvidencePath: []string{owner, cue.CueCode}, DuplicateSequence: cue.SequenceNo}
		}
		sequenceOwner[cue.SequenceNo] = cue.CueCode
		byID[cue.ID] = cue
	}
	for _, cue := range cues {
		seenDependency := map[uint]bool{}
		for _, dependencyID := range cue.DependencyIDs {
			if seenDependency[dependencyID] {
				return nil, &GraphError{Code: "DUPLICATE_DEPENDENCY", Message: fmt.Sprintf("cue %s repeats dependency %d", cue.CueCode, dependencyID), EvidencePath: []string{cue.CueCode}}
			}
			seenDependency[dependencyID] = true
			dependency, exists := byID[dependencyID]
			if !exists {
				return nil, &GraphError{Code: "MISSING_CUE_DEPENDENCY", Message: fmt.Sprintf("cue %s requires missing cue id %d", cue.CueCode, dependencyID), EvidencePath: []string{cue.CueCode, fmt.Sprintf("missing:%d", dependencyID)}, MissingDependencyID: dependencyID}
			}
			if dependency.ID == cue.ID {
				return nil, &GraphError{Code: "CUE_DEPENDENCY_CYCLE", Message: "cue cannot depend on itself", EvidencePath: []string{cue.CueCode, cue.CueCode}}
			}
		}
	}
	if path := findCycle(cues, byID); len(path) > 0 {
		return nil, &GraphError{Code: "CUE_DEPENDENCY_CYCLE", Message: "cue dependency graph contains a cycle", EvidencePath: path}
	}
	return topologicalOrder(cues, byID), nil
}

func findCycle(cues []GraphCue, byID map[uint]GraphCue) []string {
	state := make(map[uint]uint8, len(cues))
	stack := make([]uint, 0, len(cues))
	position := make(map[uint]int, len(cues))
	var visit func(uint) []string
	visit = func(id uint) []string {
		state[id] = 1
		position[id] = len(stack)
		stack = append(stack, id)
		cue := byID[id]
		for _, dependencyID := range cue.DependencyIDs {
			if state[dependencyID] == 0 {
				if path := visit(dependencyID); len(path) > 0 {
					return path
				}
			} else if state[dependencyID] == 1 {
				start := position[dependencyID]
				path := make([]string, 0, len(stack)-start+1)
				for _, cycleID := range stack[start:] {
					path = append(path, byID[cycleID].CueCode)
				}
				path = append(path, byID[dependencyID].CueCode)
				return path
			}
		}
		stack = stack[:len(stack)-1]
		delete(position, id)
		state[id] = 2
		return nil
	}
	ordered := append([]GraphCue(nil), cues...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].SequenceNo < ordered[j].SequenceNo })
	for _, cue := range ordered {
		if state[cue.ID] == 0 {
			if path := visit(cue.ID); len(path) > 0 {
				return path
			}
		}
	}
	return nil
}

func topologicalOrder(cues []GraphCue, byID map[uint]GraphCue) []GraphCue {
	indegree := make(map[uint]int, len(cues))
	dependents := make(map[uint][]uint, len(cues))
	for _, cue := range cues {
		indegree[cue.ID] = len(cue.DependencyIDs)
		for _, dependencyID := range cue.DependencyIDs {
			dependents[dependencyID] = append(dependents[dependencyID], cue.ID)
		}
	}
	ready := make([]GraphCue, 0, len(cues))
	for _, cue := range cues {
		if indegree[cue.ID] == 0 {
			ready = append(ready, cue)
		}
	}
	sortCues(ready)
	result := make([]GraphCue, 0, len(cues))
	for len(ready) > 0 {
		current := ready[0]
		ready = ready[1:]
		result = append(result, current)
		for _, dependentID := range dependents[current.ID] {
			indegree[dependentID]--
			if indegree[dependentID] == 0 {
				ready = append(ready, byID[dependentID])
				sortCues(ready)
			}
		}
	}
	return result
}

func sortCues(cues []GraphCue) {
	sort.Slice(cues, func(i, j int) bool {
		if cues[i].SequenceNo == cues[j].SequenceNo {
			return cues[i].CueCode < cues[j].CueCode
		}
		return cues[i].SequenceNo < cues[j].SequenceNo
	})
}
