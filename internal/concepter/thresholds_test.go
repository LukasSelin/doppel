package concepter

import "testing"

func TestClassifyRoleAt(t *testing.T) {
	th := RoleThresholds{FanIn: 3, FanOut: 2}
	tests := []struct {
		name     string
		callers  int
		callees  int
		wantRole string
	}{
		{"below both", 2, 1, RoleLeaf},
		{"at raised fan-in threshold", 3, 0, RoleUtility},
		{"below raised fan-in, at fan-out", 2, 2, RoleOrchestrator},
		{"at both", 3, 2, RolePassthrough},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyRoleAt(tt.callers, tt.callees, th); got != tt.wantRole {
				t.Errorf("ClassifyRoleAt(%d, %d, %+v) = %q, want %q",
					tt.callers, tt.callees, th, got, tt.wantRole)
			}
		})
	}
}

func TestClassifyRoleUsesDefaultThresholds(t *testing.T) {
	// The fixed-threshold entry point and the parameterized one must agree at
	// the defaults — ClassifyRole is the contract role_test.go pins.
	for callers := 0; callers <= 3; callers++ {
		for callees := 0; callees <= 3; callees++ {
			fixed := ClassifyRole(callers, callees)
			param := ClassifyRoleAt(callers, callees, DefaultRoleThresholds())
			if fixed != param {
				t.Errorf("ClassifyRole(%d,%d)=%q but ClassifyRoleAt(defaults)=%q",
					callers, callees, fixed, param)
			}
		}
	}
}

func TestThresholdsFromDegrees(t *testing.T) {
	tests := []struct {
		name    string
		fanIn   []int
		fanOut  []int
		wantIn  int
		wantOut int
	}{
		{
			// Median degree <= 1 on both axes: the floor holds and behavior is
			// identical to the historical fixed threshold. This is the normal
			// case — resolved internal degrees are sparse on most repos.
			name:  "sparse corpus degenerates to the floor",
			fanIn: []int{0, 0, 0, 1, 1}, fanOut: []int{0, 1, 1, 1, 2},
			wantIn: 2, wantOut: 2,
		},
		{
			// A dense graph raises the bar: median fan-out 3 means "calls three
			// repo functions" is unremarkable here, so high starts at 4.
			name:  "dense corpus raises the threshold",
			fanIn: []int{0, 1, 2, 2, 3}, fanOut: []int{2, 3, 3, 4, 5},
			wantIn: 3, wantOut: 4,
		},
		{
			name:  "empty corpus",
			fanIn: nil, fanOut: nil,
			wantIn: 2, wantOut: 2,
		},
		{
			// Zero-degree units are part of the population and drag the median
			// down; a corpus that is mostly isolated functions keeps the floor
			// even when a few hubs exist.
			name:  "zeros included in the median",
			fanIn: []int{0, 0, 0, 0, 9, 9}, fanOut: []int{0, 0, 0, 0, 9, 9},
			wantIn: 2, wantOut: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ThresholdsFromDegrees(tt.fanIn, tt.fanOut)
			if got.FanIn != tt.wantIn || got.FanOut != tt.wantOut {
				t.Errorf("ThresholdsFromDegrees = %+v, want {FanIn:%d FanOut:%d}",
					got, tt.wantIn, tt.wantOut)
			}
		})
	}
}

func TestThresholdsFromDegreesDoesNotMutateInput(t *testing.T) {
	fanIn := []int{5, 1, 3}
	ThresholdsFromDegrees(fanIn, nil)
	if fanIn[0] != 5 || fanIn[1] != 1 || fanIn[2] != 3 {
		t.Errorf("input slice was reordered: %v", fanIn)
	}
}
