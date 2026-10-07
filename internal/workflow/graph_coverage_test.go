package controller

import "testing"

func TestValidateRequiredIdentityAndDuplicateIDs(t *testing.T) {
	tests := []struct {
		name       string
		definition Definition
		wantError  string
	}{
		{name: "missing workflow name", definition: Definition{}, wantError: "workflow name is required"},
		{name: "workflow with no nodes", definition: Definition{Name: "empty"}},
		{name: "missing node id", definition: Definition{Name: "invalid", Nodes: []Node{{Name: "node"}}}, wantError: "every node requires an id and name"},
		{name: "missing node name", definition: Definition{Name: "invalid", Nodes: []Node{{ID: "node"}}}, wantError: "every node requires an id and name"},
		{name: "duplicate node id", definition: Definition{Name: "invalid", Nodes: []Node{{ID: "node", Name: "first"}, {ID: "node", Name: "second"}}}, wantError: `duplicate node id "node"`},
		{name: "optional types omitted", definition: Definition{Name: "valid", Nodes: []Node{{ID: "source", Name: "source", Output: "json"}, {ID: "sink", Name: "sink"}}, Edges: []Edge{{Source: "source", Target: "sink"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.definition)
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantError {
				t.Fatalf("Validate() error = %v, want %q", err, tt.wantError)
			}
		})
	}
}
