package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/naoina/toml"
	"github.com/pkg/errors"
)

var cooldown int64

var cfg struct {
	Token string
}

// we can get away with the following because the bot only ever gets one url.
var reallyLazyArrayUnmarshal = strings.NewReplacer(
	"[", "",
	"\"", "",
	"]", "",
)

var commands = []api.CreateCommandData{
	{
		Name:        "fox",
		Description: "fox",
	},
}

func main() {
	// Get the config file
	configFile, err := os.ReadFile("config.toml")
	if err != nil {
		log.Fatalln(err)
	}

	// Unmarshal the contents into the global config object
	err = toml.Unmarshal([]byte(configFile), &cfg)
	if err != nil {
		log.Fatalln(err)
	}

	var h handler
	h.s = state.New("Bot " + cfg.Token)
	h.s.AddInteractionHandler(&h)
	h.s.AddIntents(gateway.IntentGuilds)
	h.s.AddHandler(func(*gateway.ReadyEvent) {
		me, _ := h.s.Me()
		log.Println("connected to the gateway as", me.Tag())
	})

	if err := overwriteCommands(h.s); err != nil {
		log.Fatalln("cannot update commands:", err)
	}

	if err := h.s.Connect(context.Background()); err != nil {
		log.Fatalln("cannot connect:", err)
	}
}

func overwriteCommands(s *state.State) error {
	app, err := s.CurrentApplication()
	if err != nil {
		return errors.Wrap(err, "cannot get current app ID")
	}

	_, err = s.BulkOverwriteCommands(app.ID, commands)
	return err
}

type handler struct {
	s *state.State
}

func (h *handler) HandleInteraction(ev *discord.InteractionEvent) *api.InteractionResponse {
	switch data := ev.Data.(type) {
	case *discord.CommandInteraction:
		switch data.Name {
		case "fox":
			return h.cmdFox(ev, data)
		default:
			return errorResponse(fmt.Errorf("unknown command %q", data.Name))
		}
	default:
		return errorResponse(fmt.Errorf("unknown interaction %T", ev.Data))
	}
}

func (h *handler) cmdFox(ev *discord.InteractionEvent, _ *discord.CommandInteraction) *api.InteractionResponse {
	now := time.Now().UnixMilli()
	fmt.Printf("%v >= %v is %v\n", cooldown, now, cooldown >= now)
	if cooldown >= now {
		return &api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{
				Content: option.NewNullableString("The command has a global cooldown of 10 seconds as a curtousy, since it uses a service that the ioi does not host."),
			},
		}
	}
	// this is semantically wrong but idc it works
	cooldown = now + int64(time.Microsecond*10)

	apiURL := "https://api.fox.pics/v1/get-random-foxes?amount=1"
	resp, err := http.Get(apiURL)
	if err != nil {
		return &api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{
				Content: option.NewNullableString(err.Error()),
			},
		}
	}

	buf := make([]byte, 256)
	resp.Body.Read(buf)
	url := reallyLazyArrayUnmarshal.Replace(string(buf))

	return &api.InteractionResponse{
		Type: api.MessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Content: option.NewNullableString(url),
		},
	}
}

func errorResponse(err error) *api.InteractionResponse {
	return &api.InteractionResponse{
		Type: api.MessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Content:         option.NewNullableString("**Error:** " + err.Error()),
			Flags:           discord.EphemeralMessage,
			AllowedMentions: &api.AllowedMentions{ /* none */ },
		},
	}
}

func deferResponse(flags discord.MessageFlags) *api.InteractionResponse {
	return &api.InteractionResponse{
		Type: api.DeferredMessageInteractionWithSource,
		Data: &api.InteractionResponseData{
			Flags: flags,
		},
	}
}
