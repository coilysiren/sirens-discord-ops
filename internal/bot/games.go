package bot

// Game declares one game's coily passthrough surface. The bot pins one panel
// per Game in the admin channel, one button per verb. v1 ships eco only.
type Game struct {
	// Name appears in audit-channel prefixes and button labels (e.g. "eco").
	Name string
	// CoilyPrefix is prepended to the verb. For eco it is {"gaming","eco"},
	// so a Restart button runs `coily gaming eco restart`.
	CoilyPrefix []string
	// Verbs are the buttons rendered on the pinned message, in order.
	Verbs []string
	// ConfirmVerbs require a second click to run. The first click shows an
	// ephemeral Confirm / Cancel prompt. Only Confirm runs the verb.
	ConfirmVerbs []string
}

// needsConfirm reports whether the given verb requires a confirmation step.
func (g Game) needsConfirm(verb string) bool {
	for _, v := range g.ConfirmVerbs {
		if v == verb {
			return true
		}
	}
	return false
}

// games is the v1 registry. Edit here to add a new game.
var Games = []Game{
	{
		Name:         "eco",
		CoilyPrefix:  []string{"gaming", "eco"},
		Verbs:        []string{"restart", "status", "stop", "start"},
		ConfirmVerbs: []string{"restart", "stop", "start"},
	},
}
