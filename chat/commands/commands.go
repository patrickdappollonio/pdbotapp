package commands

import "time"

type Command struct {
	Command      string
	ModsOnly     bool
	StreamerOnly bool
	ForUsers     []string
	Cooldown     time.Duration
	Action       func(sender, message string) []string
}

var All = []Command{alive, say}
