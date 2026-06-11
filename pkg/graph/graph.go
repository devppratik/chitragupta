package graph

import (
	"fmt"
	"strings"

	"github.com/ppanda/chitragupta/pkg/semver"
	"github.com/ppanda/chitragupta/pkg/types"
)

// Node represents a package in dependency graph
type Node struct {
	Name         string
	Version      string
	Source       string
	Dependencies []*Node
	Dependents   []*Node
}

// Graph represents dependency graph
type Graph struct {
	Nodes map[string]*Node
	Root  *Node
}

// NewGraph creates empty graph
func NewGraph() *Graph {
	return &Graph{
		Nodes: make(map[string]*Node),
	}
}

// AddNode adds package to graph
func (g *Graph) AddNode(name, version, source string) *Node {
	key := nodeKey(name, version)

	if existing, ok := g.Nodes[key]; ok {
		return existing
	}

	node := &Node{
		Name:         name,
		Version:      version,
		Source:       source,
		Dependencies: make([]*Node, 0),
		Dependents:   make([]*Node, 0),
	}

	g.Nodes[key] = node
	return node
}

// AddEdge creates dependency relationship
func (g *Graph) AddEdge(from, to *Node) {
	// Add dependency
	if !contains(from.Dependencies, to) {
		from.Dependencies = append(from.Dependencies, to)
	}

	// Add dependent
	if !contains(to.Dependents, from) {
		to.Dependents = append(to.Dependents, from)
	}
}

// Resolve builds dependency graph
func (g *Graph) Resolve(rootPkg *types.Package, resolver func(name, version string) (*types.Package, error)) error {
	root := g.AddNode(rootPkg.Name, rootPkg.Version, rootPkg.Scope)
	g.Root = root

	visited := make(map[string]bool)
	return g.resolveNode(root, rootPkg, resolver, visited)
}

// resolveNode recursively resolves dependencies
func (g *Graph) resolveNode(node *Node, pkg *types.Package, resolver func(name, version string) (*types.Package, error), visited map[string]bool) error {
	key := nodeKey(node.Name, node.Version)

	if visited[key] {
		return nil // Already processed
	}

	visited[key] = true

	// Resolve dependencies
	for depName, depVersion := range pkg.Manifest.Dependencies {
		depPkg, err := resolver(depName, depVersion)
		if err != nil {
			return fmt.Errorf("failed to resolve %s@%s: %w", depName, depVersion, err)
		}

		depNode := g.AddNode(depPkg.Name, depPkg.Version, depPkg.Scope)
		g.AddEdge(node, depNode)

		// Recurse
		if err := g.resolveNode(depNode, depPkg, resolver, visited); err != nil {
			return err
		}
	}

	return nil
}

// TopologicalSort returns packages in install order
func (g *Graph) TopologicalSort() ([]*Node, error) {
	visited := make(map[string]bool)
	stack := make([]*Node, 0)
	visiting := make(map[string]bool)

	var visit func(*Node) error
	visit = func(node *Node) error {
		key := nodeKey(node.Name, node.Version)

		if visiting[key] {
			return fmt.Errorf("circular dependency detected: %s", node.Name)
		}

		if visited[key] {
			return nil
		}

		visiting[key] = true

		for _, dep := range node.Dependencies {
			if err := visit(dep); err != nil {
				return err
			}
		}

		visiting[key] = false
		visited[key] = true
		stack = append(stack, node)

		return nil
	}

	if g.Root == nil {
		return stack, nil
	}

	if err := visit(g.Root); err != nil {
		return nil, err
	}

	return stack, nil
}

// Why returns dependency chain to package
func (g *Graph) Why(targetName string) []string {
	if g.Root == nil {
		return nil
	}

	var findPath func(*Node, []string) []string
	findPath = func(node *Node, path []string) []string {
		currentPath := append(path, node.Name)

		if node.Name == targetName {
			return currentPath
		}

		for _, dep := range node.Dependencies {
			if result := findPath(dep, currentPath); result != nil {
				return result
			}
		}

		return nil
	}

	return findPath(g.Root, []string{})
}

// Deduplicate merges duplicate nodes with same name
func (g *Graph) Deduplicate() {
	// Track highest version per package name
	latest := make(map[string]*Node)

	for key, node := range g.Nodes {
		name := node.Name

		if existing, ok := latest[name]; ok {
			// Compare versions using semver
			nodeVer, err1 := semver.Parse(node.Version)
			existingVer, err2 := semver.Parse(existing.Version)

			// Fallback to string comparison if semver parsing fails
			var nodeIsNewer bool
			if err1 == nil && err2 == nil {
				nodeIsNewer = nodeVer.Compare(existingVer) > 0
			} else {
				nodeIsNewer = node.Version > existing.Version
			}

			if nodeIsNewer {
				// Remove old version
				delete(g.Nodes, nodeKey(existing.Name, existing.Version))
				latest[name] = node

				// Remap dependencies
				g.remapDependencies(existing, node)
			} else {
				// Remove current (older) version
				delete(g.Nodes, key)
			}
		} else {
			latest[name] = node
		}
	}
}

// remapDependencies updates references from old to new node
func (g *Graph) remapDependencies(old, new *Node) {
	for _, node := range g.Nodes {
		for i, dep := range node.Dependencies {
			if dep == old {
				node.Dependencies[i] = new
			}
		}
		for i, dependent := range node.Dependents {
			if dependent == old {
				node.Dependents[i] = new
			}
		}
	}
}

// String generates ASCII tree visualization
func (g *Graph) String() string {
	if g.Root == nil {
		return "(empty)"
	}

	var sb strings.Builder
	g.printNode(&sb, g.Root, "", true, make(map[string]bool))
	return sb.String()
}

// printNode recursively prints tree
func (g *Graph) printNode(sb *strings.Builder, node *Node, prefix string, isLast bool, printed map[string]bool) {
	key := nodeKey(node.Name, node.Version)

	// Print current node
	marker := "├── "
	if isLast {
		marker = "└── "
	}

	sb.WriteString(prefix)
	sb.WriteString(marker)
	sb.WriteString(fmt.Sprintf("%s@%s", node.Name, node.Version))

	if printed[key] {
		sb.WriteString(" (already shown)\n")
		return
	}

	sb.WriteString("\n")
	printed[key] = true

	// Print dependencies
	for i, dep := range node.Dependencies {
		childPrefix := prefix
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}

		g.printNode(sb, dep, childPrefix, i == len(node.Dependencies)-1, printed)
	}
}

func nodeKey(name, version string) string {
	return fmt.Sprintf("%s@%s", name, version)
}

func contains(nodes []*Node, target *Node) bool {
	for _, n := range nodes {
		if n == target {
			return true
		}
	}
	return false
}
