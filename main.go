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
	initDb()

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

		if i.Type == discordgo.InteractionModalSubmit {
			if strings.HasPrefix(i.ModalSubmitData().CustomID, verifyTokenModalPrefix) {
				handleVerifyTokenModalSubmit(s, i)
			}
			return
		}

		if i.Type == discordgo.InteractionMessageComponent {
			customID := i.MessageComponentData().CustomID
			switch {
			case strings.HasPrefix(customID, clanMemSortPrefix):
				handleClanMembersSortSelect(s, i)
			case strings.HasPrefix(customID, clanMemPrefix):
				handleClanMembersButton(s, i)
			case strings.HasPrefix(customID, clanWarSortPrefix):
				handleClanWarSortSelect(s, i)
			case strings.HasPrefix(customID, clanWarPrefix):
				handleClanWarButton(s, i)
			case strings.HasPrefix(customID, clanTabPrefix):
				handleClanTabButton(s, i)
			case strings.HasPrefix(customID, playerTabPrefix):
				handlePlayerTabButton(s, i)
			case strings.HasPrefix(customID, playerAchSortPrefix):
				handlePlayerAchievementsSortSelect(s, i)
			case strings.HasPrefix(customID, playerAchPrefix):
				handlePlayerAchievementsButton(s, i)
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

	if s.State != nil && s.State.User != nil {
		botAvatarURL = s.State.User.AvatarURL("256")
	}

	appID := discordApplicationId
	if appID == "" {
		appID = s.State.User.ID
	}

	if discordGuildID == "" {
		log.Fatal("DISCORD_GUILD_ID is required: set it to your dev server ID for guild-scoped slash commands")
	}

	// Clear global commands so only dev-guild commands from this project are visible.
	_, err = s.ApplicationCommandBulkOverwrite(appID, "", []*discordgo.ApplicationCommand{})
	if err != nil {
		log.Printf("warning: clear global slash commands: %v", err)
	} else {
		log.Println("cleared global slash commands")
	}

	_, err = s.ApplicationCommandBulkOverwrite(appID, discordGuildID, Commands)
	if err != nil {
		log.Fatalf("slash command bulk overwrite (guild %s): %v", discordGuildID, err)
	}
	log.Printf("synced %d slash command(s) on guild %s", len(Commands), discordGuildID)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	log.Println("Gracefully shutting down.")
}
