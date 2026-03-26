package concepter

import "testing"

func TestClassifyRole(t *testing.T) {
	tests := []struct {
		name     string
		callers  int
		callees  int
		wantRole string
	}{
		{"zero/zero", 0, 0, RoleLeaf},
		{"one/one", 1, 1, RoleLeaf},
		{"high callers only", 3, 0, RoleUtility},
		{"threshold callers only", 2, 1, RoleUtility},
		{"high callees only", 0, 3, RoleOrchestrator},
		{"threshold callees only", 1, 2, RoleOrchestrator},
		{"both high", 5, 5, RolePassthrough},
		{"both at threshold", 2, 2, RolePassthrough},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyRole(tt.callers, tt.callees)
			if got != tt.wantRole {
				t.Errorf("ClassifyRole(%d, %d) = %q, want %q",
					tt.callers, tt.callees, got, tt.wantRole)
			}
		})
	}
}
