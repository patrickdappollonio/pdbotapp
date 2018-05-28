package chat

import (
	"log"
	"strings"
	"time"

	"github.com/patrickdappollonio/env"
	"github.com/patrickdappollonio/pdbotapp/chat/commands"
	cache "github.com/patrickmn/go-cache"
	irc "github.com/thoj/go-ircevent"
)

const (
	twitchUsername = "pdbot"
	twitchServer   = "irc.chat.twitch.tv:443"

	modsPrefix = "The moderators of this channel are: "
)

type msgOut struct {
	Message  string
	Trusted  bool
	Cooldown time.Duration
}

var (
	// Configuration variables
	twitchToken   = env.GetDefault("TWITCH_TOKEN", "oauth:eunsq7qz3fyq1ofpn92rkh0vt152tx")
	twitchChannel = env.GetDefault("TWITCH_CHANNEL", "patrickdappollonio")
	twitchDebug   = env.GetDefault("TWITCH_DEBUG", "")

	// Mods and mutex
	twitchMods = []string{twitchChannel, twitchUsername}

	chOutgoing = make(chan msgOut)
	chIncoming = make(chan *irc.Event)
)

var cl *irc.Connection

func ircChannel() string {
	return "#" + twitchChannel
}

func Setup() error {
	// Set up connection options as well as debugging in
	// standard output.
	cl = irc.IRC(twitchUsername, twitchUsername)
	cl.Debug = twitchDebug != ""
	cl.UseTLS = true
	cl.Password = twitchToken

	// Asynchronously connect, if there was an error connecting
	// then return right away
	if err := cl.Connect(twitchServer); err != nil {
		return err
	}

	// All messages coming in from the chat window will be sent to
	// the channel of incoming messages
	cl.AddCallback("PRIVMSG", func(event *irc.Event) {
		chIncoming <- event
	})

	// Register an event for when we're successfully connected
	// to twitch, which is an IRC code 376
	cl.AddCallback("376", func(event *irc.Event) {
		log.Println("--- Connected to Twitch!")

		// Request to receive notices
		cl.SendRaw("CAP REQ :twitch.tv/commands")

		// This will queue after the connection happens, and it'll
		// send a request to join the channel specified in TWITCH_CHANNEL
		cl.Join(ircChannel())

		// Request a list of mods
		chOutgoing <- msgOut{
			Message: "/mods",
			Trusted: true,
		}
	})

	// Update mods list
	cl.AddCallback("NOTICE", func(event *irc.Event) {
		if strings.HasPrefix(event.Message(), modsPrefix) {
			twitchMods = strings.Split(strings.TrimPrefix(event.Message(), modsPrefix), ", ")
			log.Println("--- Changing mods to:", strings.Join(twitchMods, ", "))
		}
	})

	// Run the loop so we can receive
	// new messages, and also loop through the outgoing
	// connections
	go cl.Loop()
	go queueOutgoing()
	go handleIncoming()

	return nil
}

func queueOutgoing() {
	// Keep track of the last messages sent
	cached := cache.New(5*time.Minute, 5*time.Minute)

	// An infinite for loop...
	for {
		select {
		// That starts looking for outgoing messages...
		case m, ok := <-chOutgoing:
			// If the channel is closed, simply exit
			// this part
			if !ok {
				return
			}

			// If we were disconnected, we make only ONE attempt to
			// re-connect again, since we may have lost the connection
			if !cl.Connected() {
				log.Println("Attempting to send a message through a channel not connected to")
				log.Println("Logging in...")

				// Handle in case of an error
				if err := Setup(); err != nil {
					log.Fatal("Unable to re-connect to the chat. Exiting app...")
				}
			}

			// Check if we already sent this message, and if so
			// silently omit it
			if !m.Trusted {
				// If the command has a cooldown
				if m.Cooldown.Seconds() > 0 {
					// Find it in the cache...
					if _, found := cached.Get(m.Message); found {
						log.Println("Cached response found requested by untrusted user, ignoring response and moving on.")
						continue
					}

					// ... or if it wasn't found, save it to the cache
					cached.Set(m.Message, struct{}{}, m.Cooldown)
				}
			}

			// Once we're connected, we send the message to the channel
			// specified in TWITCH_CHANNEL, but first let's check if
			// the message is a command message.
			prefix := "/me "
			if len(m.Message) > 1 && m.Message[0] == '/' {
				prefix = ""
			}

			// Then send the message...
			cl.SendRawf("PRIVMSG %s :%s%s", ircChannel(), prefix, m.Message)

			// After sending a message, wait 150 milliseconds before
			// sending another one, so we never hit the limits
			time.Sleep(150 * time.Millisecond)
		default:
			// Do nothing
		}
	}
}

func handleIncoming() {
	// Perform command mapping
	cmds := make(map[string]commands.Command)
	for _, v := range commands.All {
		cmds[v.Command] = v
	}

	// Another infinite for loop
	for {
		select {
		// Fetch the message and the channel status
		case msg, ok := <-chIncoming:
			// Check if the channel is being closed
			if !ok {
				return
			}

			// If the message is a command, handle it here
			if m, isCommand := parsecmd(msg.Message()); isCommand {
				// Iterate over all available commands
				if c, found := cmds[m]; found {
					// Holder to know if it's a user we know
					isTrusted := false

					// Set this to true if it's a mod or the streamer
					if strings.ToLower(msg.User) == twitchChannel {
						isTrusted = true
					}
					if findInSlice(strings.ToLower(msg.User), twitchMods) {
						isTrusted = true
					}

					// Check if the current user is trying to execute
					// a streamer-only command, and if so, ignore it
					if c.StreamerOnly {
						if strings.ToLower(msg.User) != twitchChannel {
							log.Printf("--* User %q attempted to run streamer-only command %q but it was denied", msg.User, m)
							continue
						} else {
							log.Println("Declaring user", msg.User, "as trusted because it's the streamer")
							isTrusted = true
						}
					}

					// Check if it's a mod-only command and validate if the
					// user is a mod
					if c.ModsOnly {
						// Iterate over all the mods in the list
						// and see if the user is a mod
						isMod := findInSlice(strings.ToLower(msg.User), twitchMods)

						// If the user isn't a mod, silently decline it
						if !isMod {
							log.Printf("--* User %q attempted to run mod-only command %q but it was denied", msg.User, m)
							continue
						} else {
							log.Println("Declaring user", msg.User, "as trusted because it's a mod")
							isTrusted = true
						}
					}

					// Check if the command is for an specific user
					if len(c.ForUsers) > 0 {
						// Check if the current user is in this list
						isForItself := findInSlice(strings.ToLower(msg.User), c.ForUsers)

						// If it's not for him, cancel it
						if !isForItself {
							log.Printf("--* User %q attempted to run command %q but it was denied, command not meant for him, but meant for %s", msg.User, m, strings.Join(c.ForUsers, ", "))
							continue
						} else {
							log.Println("Declaring user", msg.User, "as trusted because it owns the command")
							isTrusted = true
						}
					}

					// Fetch all output messages and send them down the chain
					output := c.Action(msg.User, msg.Message())
					for _, v := range output {
						if len(v) > 0 {
							chOutgoing <- msgOut{
								Message:  v,
								Trusted:  isTrusted,
								Cooldown: c.Cooldown,
							}
						} else {
							log.Println("Executed empty message on command", c.Command)
						}

					}
				} else {
					log.Printf("Command %q not found", m)
				}
			}

		default:

		}
	}
}

func parsecmd(msg string) (string, bool) {
	if !strings.HasPrefix(msg, "!") {
		return "", false
	}

	firstSpace := strings.Index(msg, " ")
	if firstSpace == -1 {
		return msg[1:], true
	}

	return msg[1:firstSpace], true
}

func findInSlice(val string, slice []string) bool {
	for _, v := range slice {
		if val == v {
			return true
		}
	}

	return false
}
