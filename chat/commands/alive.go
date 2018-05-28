package commands

var alive = Command{
	Command:  "ping",
	ModsOnly: true,
	Action: func(sender, message string) []string {
		return []string{"SeemsGood I'm alive, " + sender + "!"}
	},
}
