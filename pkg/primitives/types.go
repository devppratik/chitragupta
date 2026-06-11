package primitives

// PrimitiveType represents APM primitive categories
type PrimitiveType string

const (
	TypeSkill       PrimitiveType = "skill"
	TypePrompt      PrimitiveType = "prompt"
	TypeInstruction PrimitiveType = "instruction"
	TypeAgent       PrimitiveType = "agent"
	TypeHook        PrimitiveType = "hook"
	TypeMCP         PrimitiveType = "mcp"
	TypePlugin      PrimitiveType = "plugin"
)

// Primitive represents a discovered primitive
type Primitive struct {
	Type        PrimitiveType
	Name        string
	FilePath    string
	Description string
	Content     string
	Source      string // "local" or "dependency:pkg-name"
}

// PrimitiveSet holds all discovered primitives
type PrimitiveSet struct {
	Skills       []Primitive
	Prompts      []Primitive
	Instructions []Primitive
	Agents       []Primitive
	Hooks        []Primitive
	MCPServers   []Primitive
	Plugins      []Primitive
}

// Add adds primitive to appropriate set
func (ps *PrimitiveSet) Add(p Primitive) {
	switch p.Type {
	case TypeSkill:
		ps.Skills = append(ps.Skills, p)
	case TypePrompt:
		ps.Prompts = append(ps.Prompts, p)
	case TypeInstruction:
		ps.Instructions = append(ps.Instructions, p)
	case TypeAgent:
		ps.Agents = append(ps.Agents, p)
	case TypeHook:
		ps.Hooks = append(ps.Hooks, p)
	case TypeMCP:
		ps.MCPServers = append(ps.MCPServers, p)
	case TypePlugin:
		ps.Plugins = append(ps.Plugins, p)
	}
}

// All returns all primitives
func (ps *PrimitiveSet) All() []Primitive {
	all := make([]Primitive, 0)
	all = append(all, ps.Skills...)
	all = append(all, ps.Prompts...)
	all = append(all, ps.Instructions...)
	all = append(all, ps.Agents...)
	all = append(all, ps.Hooks...)
	all = append(all, ps.MCPServers...)
	all = append(all, ps.Plugins...)
	return all
}
