package main

import (
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var (
	cocToken             string
	discordToken         string
	discordApplicationId string
	discordPublicKey     string
	discordGuildID       string
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	cocToken = strings.TrimSpace(os.Getenv("COC_TOKEN"))
	discordToken = os.Getenv("DISCORD_TOKEN")
	discordApplicationId = os.Getenv("DISCORD_APPLICATION_ID")
	discordPublicKey = os.Getenv("DISCORD_PUBLIC_KEY")
	discordGuildID = os.Getenv("DISCORD_GUILD_ID")
}

func main() {
	s, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		log.Fatal(err)
	}

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
			if h, ok := CommandAutocompleteHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
			return
		}

		if h, ok := CommandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	err = s.Open()
	if err != nil {
		log.Fatal("error opening connection:", err)
	}
	defer s.Close()

	appID := discordApplicationId
	if appID == "" {
		appID = s.State.User.ID
	}

	if discordGuildID == "" {
		log.Fatal("DISCORD_GUILD_ID is required: set it to your dev server ID for guild-scoped slash commands")
	}

	_, err = s.ApplicationCommandBulkOverwrite(appID, discordGuildID, Commands)
	if err != nil {
		log.Fatalf("slash command bulk overwrite (guild %s): %v", discordGuildID, err)
	}
	log.Printf("registered %d slash command(s) on guild %s", len(Commands), discordGuildID)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	_, err = s.ApplicationCommandBulkOverwrite(appID, discordGuildID, []*discordgo.ApplicationCommand{})
	if err != nil {
		log.Printf("warning: clear guild slash commands: %v", err)
	} else {
		log.Println("cleared guild slash commands")
	}

	log.Println("Gracefully shutting down.")
}
