package provider

// functionTogglesNeeded reports whether the is_active / is_global toggle
// endpoints must be called to move a function from its current state to the
// desired state. The toggle endpoints flip the value rather than set it.
func functionTogglesNeeded(currentActive, currentGlobal, desiredActive, desiredGlobal bool) (toggleActive, toggleGlobal bool) {
	return currentActive != desiredActive, currentGlobal != desiredGlobal
}
