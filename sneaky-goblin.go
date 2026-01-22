package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var (
	apiToken string
	botToken string
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// apiToken = os.Getenv("API_TOKEN")
	botToken = os.Getenv("BOT_TOKEN")
}

func main() {
	discord, err := discordgo.New("Bot " + botToken)
	if err != nil {
		fmt.Println("error creating Discord session", err)
		return
	}

	discord.Identify.Intents = discordgo.IntentsGuildMessages

	discord.AddHandler(interactionCreate)

	err = discord.Open()
	if err != nil {
		log.Fatal("error opening connection:", err)
	}

	cmd := &discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "Replies with Pong!",
	}

	_, err = discord.ApplicationCommandCreate(
		discord.State.User.ID,
		"", // empty = global command
		cmd,
	)
	if err != nil {
		fmt.Println("cannot create slash command:", err)
	}

	fmt.Println("Bot is running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	discord.Close()
}

func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch i.ApplicationCommandData().Name {
	case "ping":
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Pong!",
			},
		})
	}
}
