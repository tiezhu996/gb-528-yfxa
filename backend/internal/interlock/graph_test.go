package interlock

import (
	"errors"
	"reflect"
	"testing"
)

func TestValidateAndSort(t *testing.T) {
	cues := []GraphCue{
		{ID: 3, CueCode: "Q-030", SequenceNo: 30, StartOffsetMS: 3000, DurationMS: 1000, DependencyIDs: []uint{2}},
		{ID: 1, CueCode: "Q-010", SequenceNo: 10, StartOffsetMS: 0, DurationMS: 1000},
		{ID: 2, CueCode: "Q-020", SequenceNo: 20, StartOffsetMS: 1500, DurationMS: 1000, DependencyIDs: []uint{1}},
	}
	ordered, err := ValidateAndSort(cues)
	if err != nil {
		t.Fatalf("ValidateAndSort returned error: %v", err)
	}
	got := []uint{ordered[0].ID, ordered[1].ID, ordered[2].ID}
	if !reflect.DeepEqual(got, []uint{1, 2, 3}) {
		t.Fatalf("order = %v, want [1 2 3]", got)
	}
}

func TestValidateAndSortEvidence(t *testing.T) {
	tests := []struct {
		name string
		cues []GraphCue
		code string
	}{
		{"missing dependency", []GraphCue{{ID: 1, CueCode: "Q-1", SequenceNo: 1, DurationMS: 10, DependencyIDs: []uint{99}}}, "MISSING_CUE_DEPENDENCY"},
		{"duplicate sequence", []GraphCue{{ID: 1, CueCode: "Q-1", SequenceNo: 1, DurationMS: 10}, {ID: 2, CueCode: "Q-2", SequenceNo: 1, DurationMS: 10}}, "DUPLICATE_CUE_SEQUENCE"},
		{"cycle", []GraphCue{{ID: 1, CueCode: "Q-1", SequenceNo: 1, DurationMS: 10, DependencyIDs: []uint{3}}, {ID: 2, CueCode: "Q-2", SequenceNo: 2, DurationMS: 10, DependencyIDs: []uint{1}}, {ID: 3, CueCode: "Q-3", SequenceNo: 3, DurationMS: 10, DependencyIDs: []uint{2}}}, "CUE_DEPENDENCY_CYCLE"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ValidateAndSort(test.cues)
			var graphErr *GraphError
			if !errors.As(err, &graphErr) {
				t.Fatalf("expected GraphError, got %v", err)
			}
			if graphErr.Code != test.code {
				t.Fatalf("code = %q, want %q", graphErr.Code, test.code)
			}
			if len(graphErr.EvidencePath) == 0 {
				t.Fatal("expected a non-empty evidence path")
			}
			if test.code == "CUE_DEPENDENCY_CYCLE" && graphErr.EvidencePath[0] != graphErr.EvidencePath[len(graphErr.EvidencePath)-1] {
				t.Fatalf("cycle path is not closed: %v", graphErr.EvidencePath)
			}
		})
	}
}
