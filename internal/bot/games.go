package bot

// Game declares one game's coily passthrough surface. The bot pins one
// message per Game in the admin-control channel, with one button per verb.
//
// v1 registers a single Game (eco). Adding factorio later is a config edit.
type Game struct {
	// Name appears in audit-channel prefixes and button labels (e.g. "eco").
	Name string
	// CoilyPrefix is prepended to the verb when invoking coily. For eco
	// this is {"gaming", "eco"}, so a Restart button runs
	// `coily gaming eco restart`.
	CoilyPrefix []string
	// Verbs are the buttons rendered on the pinned message, in order.
	Verbs []string
	// ConfirmVerbs are the verbs that require a second click to confirm
	// before they execute. The first click shows an ephemeral prompt with
	// Confirm / Cancel buttons; only Confirm runs the verb.
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
