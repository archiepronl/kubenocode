package controller

import "fmt"

type Node struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Input    string `json:"inputType"`
	Output   string `json:"outputType"`
	Position *struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	} `json:"position,omitempty"`
}

type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type Definition struct {
	Name  string `json:"name"`
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

func Validate(definition Definition) error {
	if definition.Name == "" {
		return fmt.Errorf("workflow name is required")
	}
	nodes := make(map[string]Node, len(definition.Nodes))
	for _, node := range definition.Nodes {
		if node.ID == "" || node.Name == "" {
			return fmt.Errorf("every node requires an id and name")
		}
		if _, exists := nodes[node.ID]; exists {
			return fmt.Errorf("duplicate node id %q", node.ID)
		}
		nodes[node.ID] = node
	}
	adjacency := make(map[string][]string, len(nodes))
	for _, edge := range definition.Edges {
		source, sourceOK := nodes[edge.Source]
		target, targetOK := nodes[edge.Target]
		if !sourceOK || !targetOK {
			return fmt.Errorf("edge %q -> %q references an unknown node", edge.Source, edge.Target)
		}
		if source.Output != "" && target.Input != "" && source.Output != target.Input {
			return fmt.Errorf("node %q outputs %q but node %q expects %q", source.ID, source.Output, target.ID, target.Input)
		}
		adjacency[edge.Source] = append(adjacency[edge.Source], edge.Target)
	}

	visited := make(map[string]bool, len(nodes))
	active := make(map[string]bool, len(nodes))
	var visit func(string) error
	visit = func(id string) error {
		if active[id] {
			return fmt.Errorf("workflow contains a cycle at node %q", id)
		}
		if visited[id] {
			return nil
		}
		active[id] = true
		for _, next := range adjacency[id] {
			if err := visit(next); err != nil {
				return err
			}
		}
		active[id] = false
		visited[id] = true
		return nil
	}
	for id := range nodes {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}
