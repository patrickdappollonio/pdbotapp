package commands

import (
	"strings"
	"time"
)

var say = Command{
	Command:  "say",
	Cooldown: 5 * time.Minute,
	Action: func(sender, message string) []string {
		return []string{strings.TrimPrefix(message, "!say")}
	},
}
