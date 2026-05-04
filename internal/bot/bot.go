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
// channel if one isn't already there. Detection is by content prefix
// "sirens-discord-ops:<game>" in the message content, which the bot owns.
func (b *Bot) ensureControlPanel() error {
	msgs, err := b.session.ChannelMessages(b.cfg.AdminChannelID, 100, "", "", "")
	if err != nil {
		return fmt.Errorf("ChannelMessages: %w", err)
	}
	have := map[string]string{} // game name -> message id
	for _, m := range msgs {
		if m.Author == nil || m.Author.ID != b.session.State.User.ID {
			continue
		}
		for _, g := range b.games {
			marker := "sirens-discord-ops:" + g.Name
			if strings.HasPrefix(m.Content, marker) {
				have[g.Name] = m.ID
			}
		}
	}
	for _, g := range b.games {
		if id, ok := have[g.Name]; ok {
			// Edit in place so verb-list changes propagate without a
			// fresh post.
			_, err := b.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
				Channel:    b.cfg.AdminChannelID,
				ID:         id,
				Content:    ptr(panelContent(g)),
				Components: &[]discordgo.MessageComponent{panelRow(g)},
			})
			if err != nil {
				log.Printf("edit panel %s: %v", g.Name, err)
			}
			continue
		}
		m, err := b.session.ChannelMessageSendComplex(b.cfg.AdminChannelID, &discordgo.MessageSend{
			Content:    panelContent(g),
			Components: []discordgo.MessageComponent{panelRow(g)},
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

func panelContent(g Game) string {
	return fmt.Sprintf("sirens-discord-ops:%s\n**%s** controls", g.Name, g.Name)
}

func panelRow(g Game) discordgo.ActionsRow {
	row := discordgo.ActionsRow{}
	for _, v := range g.Verbs {
		row.Components = append(row.Components, discordgo.Button{
			Label:    fmt.Sprintf("%s %s", g.Name, v),
			Style:    buttonStyle(v),
			CustomID: fmt.Sprintf("%s:%s:%s", customIDPrefix, g.Name, v),
		})
	}
	return row
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
	parts := strings.SplitN(cid, ":", 3)
	if len(parts) != 3 || parts[0] != customIDPrefix {
		return
	}
	gameName, verb := parts[1], parts[2]
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

func (b *Bot) runVerb(i *discordgo.InteractionCreate, game Game, verb string) {
	actor := actorMention(i)
	startMsg := fmt.Sprintf("[%s] %s started %s", game.Name, actor, verb)
	if _, err := b.session.ChannelMessageSend(b.cfg.AuditChannelID, startMsg); err != nil {
		log.Printf("audit start: %v", err)
	}
	args := append(append([]string{}, game.CoilyPrefix...), verb)
	// Coily verbs are typically fast, but `restart` waits on systemctl. A
	// 5-minute ceiling is generous and prevents a stuck invocation from
	// pinning the bot.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	res, err := runCoily(ctx, b.cfg.CoilyBin, args)
	doneMsg := buildDoneMessage(game.Name, verb, res, err)
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
func buildDoneMessage(game, verb string, res CoilyResult, err error) string {
	header := fmt.Sprintf("[%s] %s complete (exit %d)", game, verb, res.ExitCode)
	if err != nil {
		header = fmt.Sprintf("[%s] %s failed to start: %v", game, verb, err)
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
