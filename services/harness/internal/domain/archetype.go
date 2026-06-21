package domain

// Archetype classifies an agent by its purpose-level operating style — how it
// behaves — independent of the model vendor.
type Archetype string

const (
	// ArchetypeCommunicator is instruction-following and mechanics-driven
	// (detailed checklists, nested workflows). Model family: Claude ▸ Kimi ▸ GLM.
	ArchetypeCommunicator Archetype = "communicator"
	// ArchetypePrincipleDriven is autonomous and goal-first: state the goal, it
	// finds the mechanics. Model family: GPT ▸ Deepseek.
	ArchetypePrincipleDriven Archetype = "principle-driven"
	// ArchetypeUtilityRunner favours speed over intelligence for cheap,
	// high-volume work. Model family: MiniMax.
	ArchetypeUtilityRunner Archetype = "utility-runner"
	// ArchetypeNone marks a model-less agent (e.g. prayer).
	ArchetypeNone Archetype = "none"
)

// Valid reports whether a is a known archetype.
func (a Archetype) Valid() bool {
	switch a {
	case ArchetypeCommunicator, ArchetypePrincipleDriven, ArchetypeUtilityRunner, ArchetypeNone:
		return true
	default:
		return false
	}
}
