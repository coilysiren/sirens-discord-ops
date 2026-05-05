package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// customIDPrefix tags interactions belonging to this bot. Format:
//
//	sdo:<game>:<verb>
//
// Discord button custom IDs are limited to 100 chars; verb and game names
// here stay well under that.
const customIDPrefix = "sdo"

// Bot wires the Discord session, configuration, and game registry.
type Bot struct {
	cfg     Config
	session *discordgo.Session
	games   []Game
}

func NewBot(cfg Config, gs []Game) (*Bot, error) {
	s, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("discordgo.New: %w", err)
	}
	// We need the GuildMembers intent to read role membership of the
	// interaction actor reliably. Interactions ship Member with roles
	// already, so this is belt-and-suspenders for edge cases.
	s.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages
	b := &Bot{cfg: cfg, session: s, games: gs}
	s.AddHandler(b.onReady)
	s.AddHandler(b.onInteraction)
	return b, nil
}

func (b *Bot) Run(ctx context.Context) error {
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("discord open: %w", err)
	}
	defer b.session.Close()
	<-ctx.Done()
	return nil
}

func (b *Bot) onReady(s *discordgo.Session, _ *discordgo.Ready) {
	log.Printf("ready as %s", s.State.User.Username)
	if err := b.ensureControlPanel(); err != nil {
		log.Printf("ensureControlPanel: %v", err)
	}
}

// ensureControlPanel posts one pinned message per game in the admin-control
// channel if one isn't already there. Panels use Components V2 so we can put
// large Separator gaps between buttons; detection walks components for our
// TextDisplay marker. Legacy V1 panels (marker in m.Content) are deleted and
// recreated as V2.
func (b *Bot) ensureControlPanel() error {
	msgs, err := b.session.ChannelMessages(b.cfg.AdminChannelID, 100, "", "", "")
	if err != nil {
		return fmt.Errorf("ChannelMessages: %w", err)
	}
	have := map[string]string{} // game name -> V2 message id
	var legacy []string         // V1 message ids to delete
	for _, m := range msgs {
		if m.Author == nil || m.Author.ID != b.session.State.User.ID {
			continue
		}
		game := detectPanelGame(m, b.games)
		if game == "" {
			continue
		}
		if m.Flags&discordgo.MessageFlagsIsComponentsV2 != 0 {
			have[game] = m.ID
		} else {
			legacy = append(legacy, m.ID)
		}
	}
	for _, id := range legacy {
		if err := b.session.ChannelMessageDelete(b.cfg.AdminChannelID, id); err != nil {
			log.Printf("delete legacy panel %s: %v", id, err)
		}
	}
	for _, g := range b.games {
		if id, ok := have[g.Name]; ok {
			_, err := b.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
				Channel:    b.cfg.AdminChannelID,
				ID:         id,
				Components: ptr(panelComponents(g)),
				Flags:      discordgo.MessageFlagsIsComponentsV2,
			})
			if err != nil {
				log.Printf("edit panel %s: %v", g.Name, err)
			}
			continue
		}
		m, err := b.session.ChannelMessageSendComplex(b.cfg.AdminChannelID, &discordgo.MessageSend{
			Components: panelComponents(g),
			Flags:      discordgo.MessageFlagsIsComponentsV2,
		})
		if err != nil {
			log.Printf("send panel %s: %v", g.Name, err)
			continue
		}
		if err := b.session.ChannelMessagePin(b.cfg.AdminChannelID, m.ID); err != nil {
			log.Printf("pin panel %s: %v", g.Name, err)
		}
	}
	return nil
}

func panelMarker(name string) string {
	return "sirens-discord-ops:" + name
}

// detectPanelGame returns the game name a panel message belongs to, or "".
// Recognizes both V2 panels (marker in a TextDisplay component) and legacy V1
// panels (marker as a content prefix).
func detectPanelGame(m *discordgo.Message, games []Game) string {
	for _, g := range games {
		marker := panelMarker(g.Name)
		if strings.HasPrefix(m.Content, marker) {
			return g.Name
		}
		for _, c := range m.Components {
			if td, ok := c.(*discordgo.TextDisplay); ok && strings.HasPrefix(td.Content, marker) {
				return g.Name
			}
			if td, ok := c.(discordgo.TextDisplay); ok && strings.HasPrefix(td.Content, marker) {
				return g.Name
			}
		}
	}
	return ""
}

// panelComponents builds the V2 component tree: a TextDisplay header carrying
// the marker, followed by one ActionsRow per verb with a large Separator
// between rows so buttons aren't packed tightly on mobile.
func panelComponents(g Game) []discordgo.MessageComponent {
	out := make([]discordgo.MessageComponent, 0, 1+2*len(g.Verbs))
	out = append(out, discordgo.TextDisplay{
		Content: fmt.Sprintf("%s\n**%s** controls", panelMarker(g.Name), g.Name),
	})
	large := discordgo.SeparatorSpacingSizeLarge
	noDivider := false
	for i, v := range g.Verbs {
		if i > 0 {
			out = append(out, discordgo.Separator{
				Spacing: &large,
				Divider: &noDivider,
			})
		}
		out = append(out, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    fmt.Sprintf("%s %s", g.Name, v),
					Style:    buttonStyle(v),
					CustomID: fmt.Sprintf("%s:%s:%s", customIDPrefix, g.Name, v),
				},
			},
		})
	}
	return out
}

func buttonStyle(verb string) discordgo.ButtonStyle {
	switch verb {
	case "stop":
		return discordgo.DangerButton
	case "start", "restart":
		return discordgo.SuccessButton
	default:
		return discordgo.SecondaryButton
	}
}

func (b *Bot) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}
	cid := i.MessageComponentData().CustomID
	parts := strings.SplitN(cid, ":", 4)
	if len(parts) < 3 || parts[0] != customIDPrefix {
		return
	}
	gameName, verb := parts[1], parts[2]
	action := ""
	if len(parts) == 4 {
		action = parts[3]
	}
	game, ok := b.findGame(gameName)
	if !ok {
		b.respondEphemeral(i, "unknown game: "+gameName)
		return
	}
	if !contains(game.Verbs, verb) {
		b.respondEphemeral(i, "unknown verb: "+verb)
		return
	}
	if !b.isAdmin(i) {
		b.respondEphemeral(i, "not authorized")
		return
	}
	switch action {
	case "":
		if game.needsConfirm(verb) {
			b.promptConfirm(i, game, verb)
			return
		}
	case "go":
		// Confirmed - fall through to run.
	case "no":
		b.updateEphemeral(i, fmt.Sprintf("cancelled `coily %s %s`", strings.Join(game.CoilyPrefix, " "), verb))
		return
	default:
		b.respondEphemeral(i, "unknown action: "+action)
		return
	}
	// Defer the response so we can take longer than 3 seconds to run
	// coily. The follow-up message is ephemeral to the actor.
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		log.Printf("defer interaction: %v", err)
		return
	}
	go b.runVerb(i, game, verb)
}

// promptConfirm sends an ephemeral confirmation prompt with Confirm / Cancel
// buttons. The Confirm button's customID carries the original game and verb
// plus the "go" action; Cancel carries "no".
func (b *Bot) promptConfirm(i *discordgo.InteractionCreate, game Game, verb string) {
	cmd := "coily " + strings.Join(append(append([]string{}, game.CoilyPrefix...), verb), " ")
	confirmID := fmt.Sprintf("%s:%s:%s:go", customIDPrefix, game.Name, verb)
	cancelID := fmt.Sprintf("%s:%s:%s:no", customIDPrefix, game.Name, verb)
	err := b.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Run `%s`?", cmd),
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Confirm",
							Style:    buttonStyle(verb),
							CustomID: confirmID,
						},
						discordgo.Button{
							Label:    "Cancel",
							Style:    discordgo.SecondaryButton,
							CustomID: cancelID,
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("prompt confirm: %v", err)
	}
}

// updateEphemeral edits the ephemeral message that hosts the component the
// user just clicked, clearing its buttons.
func (b *Bot) updateEphemeral(i *discordgo.InteractionCreate, msg string) {
	err := b.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    msg,
			Components: []discordgo.MessageComponent{},
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("update ephemeral: %v", err)
	}
}

func (b *Bot) runVerb(i *discordgo.InteractionCreate, game Game, verb string) {
	actor := actorMention(i)
	args := append(append([]string{}, game.CoilyPrefix...), verb)
	cmd := "coily " + strings.Join(args, " ")
	startMsg := fmt.Sprintf("%s started `%s`", actor, cmd)
	if _, err := b.session.ChannelMessageSend(b.cfg.AuditChannelID, startMsg); err != nil {
		log.Printf("audit start: %v", err)
	}
	// Coily verbs are typically fast, but `restart` waits on systemctl. A
	// 5-minute ceiling is generous and prevents a stuck invocation from
	// pinning the bot.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	res, err := runCoily(ctx, b.cfg.CoilyBin, args)
	doneMsg := buildDoneMessage(cmd, res, err)
	if _, sendErr := b.session.ChannelMessageSend(b.cfg.AuditChannelID, doneMsg); sendErr != nil {
		log.Printf("audit done: %v", sendErr)
	}
	follow := fmt.Sprintf("done, see <#%s>", b.cfg.AuditChannelID)
	if _, err := b.session.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Content: follow,
		Flags:   discordgo.MessageFlagsEphemeral,
	}); err != nil {
		log.Printf("followup: %v", err)
	}
}

// buildDoneMessage formats the audit-channel completion message.
//
// Discord caps message content at 2000 chars. The fenced code block plus
// header eats some of that, so we truncate coily output to fit and append
// a "(truncated)" marker. The full output is in journalctl on kai-server
// for forensics.
func buildDoneMessage(cmd string, res CoilyResult, err error) string {
	header := fmt.Sprintf("`%s` complete (exit %d)", cmd, res.ExitCode)
	if err != nil {
		header = fmt.Sprintf("`%s` failed to start: %v", cmd, err)
	}
	body := res.Output
	const maxBody = 1800
	truncated := false
	if len(body) > maxBody {
		body = body[len(body)-maxBody:]
		truncated = true
	}
	out := header + "\n```\n" + body
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	out += "```"
	if truncated {
		out += "\n(output truncated, see journalctl on kai-server)"
	}
	return out
}

func (b *Bot) findGame(name string) (Game, bool) {
	for _, g := range b.games {
		if g.Name == name {
			return g, true
		}
	}
	return Game{}, false
}

func (b *Bot) isAdmin(i *discordgo.InteractionCreate) bool {
	if i.Member == nil {
		return false
	}
	for _, r := range i.Member.Roles {
		if r == b.cfg.AdminRoleID {
			return true
		}
	}
	return false
}

func (b *Bot) respondEphemeral(i *discordgo.InteractionCreate, msg string) {
	err := b.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("respond: %v", err)
	}
}

func actorMention(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return "<@" + i.Member.User.ID + ">"
	}
	if i.User != nil {
		return "<@" + i.User.ID + ">"
	}
	return "unknown"
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func ptr[T any](v T) *T { return &v }
