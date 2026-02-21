package models

// Package represents a single npm package with version
type Package struct {
	ID      string `json:"id"`      // "lodash@4.17.21"
	Name    string `json:"name"`    // "lodash"
	Version string `json:"version"` // "4.17.21"
}

// PackageNode represents a package in the dependency graph
type PackageNode struct {
	Package
	ResolvedURL  string            `json:"resolved"`     // tarball URL
	Integrity    string            `json:"integrity"`    // sha512 hash
	Dependencies map[string]string `json:"dependencies"` // name -> version
}

// DependencyGraph represents the complete dependency tree
type DependencyGraph struct {
	RootPackage *Package                `json:"root"`
	Nodes       map[string]*PackageNode `json:"nodes"` // keyed by ID (name@version)
}

// NewDependencyGraph creates a new empty graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		Nodes: make(map[string]*PackageNode),
	}
}

// AddNode adds a package node to the graph
func (g *DependencyGraph) AddNode(node *PackageNode) {
	g.Nodes[node.ID] = node
}

// GetDirectDependencies returns the direct dependencies of the root package
func (g *DependencyGraph) GetDirectDependencies() []*PackageNode {
	if g.RootPackage == nil {
		return nil
	}

	rootNode, exists := g.Nodes[g.RootPackage.ID]
	if !exists {
		return nil
	}

	// Build name->node lookup for O(1) access
	nameToNode := make(map[string]*PackageNode)
	for _, node := range g.Nodes {
		if node.ID != g.RootPackage.ID {
			nameToNode[node.Name] = node
		}
	}

	var deps []*PackageNode
	for depName := range rootNode.Dependencies {
		if node, exists := nameToNode[depName]; exists {
			deps = append(deps, node)
		}
	}
	return deps
}
