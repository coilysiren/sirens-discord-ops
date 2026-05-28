package bot

import (
	"fmt"
	"os"
)

// Config is the runtime configuration loaded from environment variables.
// Prod fills these from SSM in start.sh. Run locally by exporting them.
type Config struct {
	Token            string
	AdminChannelID   string
	AuditChannelID   string
	AdminRoleID      string
	CoilyBin         string // path to the coily binary; defaults to "coily"
}

func LoadConfig() (Config, error) {
	c := Config{
		Token:          os.Getenv("DISCORD_TOKEN"),
		AdminChannelID: os.Getenv("ADMIN_CHANNEL_ID"),
		AuditChannelID: os.Getenv("AUDIT_CHANNEL_ID"),
		AdminRoleID:    os.Getenv("ADMIN_ROLE_ID"),
		CoilyBin:       os.Getenv("COILY_BIN"),
	}
	if c.CoilyBin == "" {
		c.CoilyBin = "coily"
	}
	missing := []string{}
	if c.Token == "" {
		missing = append(missing, "DISCORD_TOKEN")
	}
	if c.AdminChannelID == "" {
		missing = append(missing, "ADMIN_CHANNEL_ID")
	}
	if c.AuditChannelID == "" {
		missing = append(missing, "AUDIT_CHANNEL_ID")
	}
	if c.AdminRoleID == "" {
		missing = append(missing, "ADMIN_ROLE_ID")
	}
	if len(missing) > 0 {
		return c, fmt.Errorf("missing required env: %v", missing)
	}
	return c, nil
}
