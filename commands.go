package main

import (
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func getOptionMap(i *discordgo.InteractionCreate) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}
	return optionMap
}

var helpUsageByCommand = map[string]string{
	"clan":   "/clan clan-tag:<#TAG> - Displays a clan's information.",
	"help":   "/help [command] - Displays all commands or help for one command.",
	"player": "/player [player-tag:<#TAG>] - Displays a player's information.",
	"verify": "/verify player-tag:<#TAG> api-token:<TOKEN> - Links your account.",
}

func getHelpCommandNames() []string {
	names := make([]string, 0, len(helpUsageByCommand))
	for name := range helpUsageByCommand {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func getHelpCommandChoices(partial string) []*discordgo.ApplicationCommandOptionChoice {
	partial = strings.ToLower(strings.TrimSpace(partial))

	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(helpUsageByCommand))
	for _, name := range getHelpCommandNames() {
		if partial != "" && !strings.Contains(name, partial) {
			continue
		}
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  name,
			Value: name,
		})
	}
	return choices
}

func handleHelpAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var focusedValue string
	for _, option := range i.ApplicationCommandData().Options {
		if option.Name == "command" && option.Focused {
			if v, ok := option.Value.(string); ok {
				focusedValue = v
			}
			break
		}
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: getHelpCommandChoices(focusedValue),
		},
	})
}

var CommandAutocompleteHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"help": handleHelpAutocomplete,
}

var (
	Commands = []*discordgo.ApplicationCommand{
		{
			Name:        "clan",
			Description: "Displays a clan's information.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "clan-tag",
					Description: "The clan's tag.",
					Required:    true,
				},
			},
		},
		{
			Name:        "help",
			Description: "Displays a list of commands.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "command",
					Description:  "The command to display help for.",
					Required:     false,
					Autocomplete: true,
				},
			},
		},
		{
			Name:        "player",
			Description: "Displays a player's information.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "player-tag",
					Description: "The player's tag.",
					Required:    false,
				},
			},
		},
		{
			Name:        "verify",
			Description: "Links your Clash of Clans account to your Discord account.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "player-tag",
					Description: "Your Clash of Clans account tag.",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "api-token",
					Description: "Your Clash of Clans API token.",
					Required:    true,
				},
			},
		},
	}

	CommandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"clan": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			optionMap := getOptionMap(i)
			clanTag := optionMap["clan-tag"].Value.(string)
			clan, ok := getClanByTag(clanTag)
			if !ok {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Failed to fetch clan data. Please verify the tag and try again.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{
						{
							Title:       clan.Name,
							Description: clan.Description,
						},
					},
				},
			})
		},
		"help": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			optionMap := getOptionMap(i)
			embed := &discordgo.MessageEmbed{
				Footer: &discordgo.MessageEmbedFooter{
					Text: time.Now().Format("2006-01-02 15:04:05"),
				},
			}

			commandOption, exists := optionMap["command"]
			if exists && commandOption != nil {
				commandName, ok := commandOption.Value.(string)
				commandName = strings.ToLower(strings.TrimSpace(commandName))
				if ok && commandName != "" {
					usage, found := helpUsageByCommand[commandName]
					if !found {
						s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
							Type: discordgo.InteractionResponseChannelMessageWithSource,
							Data: &discordgo.InteractionResponseData{
								Content: "Unknown command. Try /help with one of: clan, help, player, verify.",
								Flags:   discordgo.MessageFlagsEphemeral,
							},
						})
						return
					}

					embed.Title = "Help: /" + commandName
					embed.Description = usage
				}
			}

			if embed.Title == "" {
				embed.Title = "Available Commands"
				fields := make([]*discordgo.MessageEmbedField, 0, len(helpUsageByCommand))
				for _, name := range getHelpCommandNames() {
					fields = append(fields, &discordgo.MessageEmbedField{
						Name:  "/" + name,
						Value: helpUsageByCommand[name],
					})
				}
				embed.Fields = fields
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
		},
		"player": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			optionMap := getOptionMap(i)
			playerOption, exists := optionMap["player-tag"]
			if !exists || playerOption == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Please provide a player tag.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			playerTag, ok := playerOption.Value.(string)
			if !ok || playerTag == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Invalid player tag. Please provide a valid tag like #ABC123.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			player, ok := getPlayerByTag(playerTag)
			if !ok {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Could not fetch that player. Double-check the tag and ensure the API token has access.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{
						{
							Title: player.Name,
						},
					},
				},
			})
		},
		"verify": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "verify",
				},
			})
		},
	}
)
