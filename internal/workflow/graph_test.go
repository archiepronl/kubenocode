package controller

import "testing"

func TestValidateRejectsCyclesAndTypeMismatches(t *testing.T) {
	base := Definition{Name: "orders", Nodes: []Node{
		{ID: "source", Name: "Receive orders", Output: "json"},
		{ID: "sink", Name: "Store orders", Input: "json"},
	}}
	if err := Validate(Definition{Name: base.Name, Nodes: base.Nodes, Edges: []Edge{{Source: "source", Target: "missing"}}}); err == nil {
		t.Fatal("expected unknown node error")
	}
	if err := Validate(Definition{Name: base.Name, Nodes: base.Nodes, Edges: []Edge{{Source: "source", Target: "sink"}}}); err != nil {
		t.Fatalf("valid graph rejected: %v", err)
	}
	base.Nodes[1].Input = "text"
	if err := Validate(Definition{Name: base.Name, Nodes: base.Nodes, Edges: []Edge{{Source: "source", Target: "sink"}}}); err == nil {
		t.Fatal("expected type mismatch error")
	}
	base.Nodes[1].Input = "json"
	if err := Validate(Definition{Name: base.Name, Nodes: base.Nodes, Edges: []Edge{{Source: "source", Target: "sink"}, {Source: "sink", Target: "source"}}}); err == nil {
		t.Fatal("expected cycle error")
	}
}
