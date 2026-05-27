package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type commandContext struct {
	subcommand string
	options    map[string]*discordgo.ApplicationCommandInteractionDataOption
}

func getCommandContext(i *discordgo.InteractionCreate) commandContext {
	data := i.ApplicationCommandData()
	if len(data.Options) == 0 {
		return commandContext{options: map[string]*discordgo.ApplicationCommandInteractionDataOption{}}
	}
	first := data.Options[0]
	if first.Type == discordgo.ApplicationCommandOptionSubCommand {
		options := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(first.Options))
		for _, opt := range first.Options {
			options[opt.Name] = opt
		}
		return commandContext{subcommand: first.Name, options: options}
	}
	options := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(data.Options))
	for _, opt := range data.Options {
		options[opt.Name] = opt
	}
	return commandContext{options: options}
}

func getFocusedAutocompleteValue(i *discordgo.InteractionCreate) (string, string) {
	data := i.ApplicationCommandData()
	if len(data.Options) == 0 {
		return "", ""
	}
	first := data.Options[0]
	if first.Type == discordgo.ApplicationCommandOptionSubCommand {
		for _, opt := range first.Options {
			if opt.Focused {
				if v, ok := opt.Value.(string); ok {
					return first.Name, v
				}
				return first.Name, ""
			}
		}
		return first.Name, ""
	}
	for _, opt := range data.Options {
		if opt.Focused {
			if v, ok := opt.Value.(string); ok {
				return "", v
			}
			return "", ""
		}
	}
	return "", ""
}

func interactionUserID(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

func stringOption(options map[string]*discordgo.ApplicationCommandInteractionDataOption, key string) string {
	opt, ok := options[key]
	if !ok || opt == nil {
		return ""
	}
	value, ok := opt.Value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func ephemeralText(s *discordgo.Session, i *discordgo.InteractionCreate, text string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: text,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func respondWithEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func playerAutocompleteChoices(discordUserID, partial string) []*discordgo.ApplicationCommandOptionChoice {
	results := searchPlayers(discordUserID, partial, 20)
	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(results))
	for _, item := range results {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  fmt.Sprintf("%s (%s)", item.Name, item.Tag),
			Value: item.Tag,
		})
	}
	return choices
}

func clanAutocompleteChoices(partial string) []*discordgo.ApplicationCommandOptionChoice {
	results := searchClans(partial, 20)
	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(results))
	for _, item := range results {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  fmt.Sprintf("%s (%s)", item.Name, item.Tag),
			Value: item.Tag,
		})
	}
	return choices
}

func resolvePlayerTag(discordUserID, input string) (string, bool) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		if mainTag, ok := getUserMainAccount(discordUserID); ok {
			return mainTag, true
		}
		return "", false
	}
	if strings.HasPrefix(raw, "#") {
		return normalizeTag(raw), true
	}
	if tag, ok := getPlayerTagByName(raw); ok {
		return normalizeTag(tag), true
	}
	matches := searchPlayers(discordUserID, raw, 1)
	if len(matches) == 0 {
		return normalizeTag(raw), true
	}
	return normalizeTag(matches[0].Tag), true
}

func resolveClanTag(discordUserID, input string) (string, bool) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		mainTag, ok := getUserMainAccount(discordUserID)
		if !ok {
			return "", false
		}
		playerResult := getPlayerByTag(mainTag)
		if !playerResult.OK || playerResult.Data.Player.Tag == "" {
			return "", false
		}
		return normalizeTag(playerResult.Data.Player.Tag), true
	}
	if strings.HasPrefix(raw, "#") {
		return normalizeTag(raw), true
	}
	if tag, ok := getClanTagByName(raw); ok {
		return normalizeTag(tag), true
	}
	matches := searchClans(raw, 1)
	if len(matches) == 0 {
		return normalizeTag(raw), true
	}
	return normalizeTag(matches[0].Tag), true
}

func makeProgressField(name string, items []PlayerItemLevel) *discordgo.MessageEmbedField {
	if len(items) == 0 {
		return &discordgo.MessageEmbedField{Name: name, Value: "No data available."}
	}
	completed := 0
	lines := make([]string, 0, len(items))
	for _, item := range items {
		if item.MaxLevel > 0 && item.Level >= item.MaxLevel {
			completed++
		}
		lines = append(lines, fmt.Sprintf("%s %d/%d", item.Name, item.Level, item.MaxLevel))
	}
	if len(lines) > 8 {
		lines = lines[:8]
	}
	percent := float64(completed) / float64(len(items)) * 100
	value := fmt.Sprintf("Completed: %d/%d (%.0f%%)\n%s", completed, len(items), percent, strings.Join(lines, "\n"))
	return &discordgo.MessageEmbedField{Name: name, Value: value}
}

func buildClanOverviewEmbed(clan Clan) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s (%s)", clan.Name, clan.Tag),
		Description: clan.Description,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Members", Value: fmt.Sprintf("%d", clan.Members), Inline: true},
			{Name: "Clan Level", Value: fmt.Sprintf("%d", clan.ClanLevel), Inline: true},
			{Name: "Clan Points", Value: fmt.Sprintf("%d", clan.ClanPoints), Inline: true},
			{Name: "War Record", Value: fmt.Sprintf("%dW / %dL / %dT", clan.WarWins, clan.WarLosses, clan.WarTies), Inline: true},
			{Name: "War Frequency", Value: string(clan.WarFrequency), Inline: true},
			{Name: "Capital Points", Value: fmt.Sprintf("%d", clan.ClanCapitalPoints), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: time.Now().Format("2006-01-02 15:04:05")},
	}
}

func buildPlayerOverviewEmbed(player Player) *discordgo.MessageEmbed {
	clanName := "No clan"
	if player.Player.Name != "" {
		clanName = player.Player.Name
	}
	return &discordgo.MessageEmbed{
		Title: fmt.Sprintf("%s (%s)", player.Name, player.Tag),
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Town Hall", Value: fmt.Sprintf("%d", player.TownHallLevel), Inline: true},
			{Name: "Trophies", Value: fmt.Sprintf("%d", player.Trophies), Inline: true},
			{Name: "War Stars", Value: fmt.Sprintf("%d", player.WarStars), Inline: true},
			{Name: "Clan", Value: clanName, Inline: true},
			{Name: "Role", Value: string(player.Role), Inline: true},
			{Name: "Donations", Value: fmt.Sprintf("%d", player.Donations), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: time.Now().Format("2006-01-02 15:04:05")},
	}
}

func buildPlayerEquipmentProgressEmbed(player Player) *discordgo.MessageEmbed {
	lines := make([]string, 0, len(player.HeroEquipment))
	for _, item := range player.HeroEquipment {
		lines = append(lines, fmt.Sprintf("%s %d/%d", item.Name, item.Level, item.MaxLevel))
	}
	if len(lines) == 0 {
		lines = append(lines, "No hero equipment found.")
	}
	if len(lines) > 15 {
		lines = lines[:15]
	}
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s Equipment Progress", player.Name),
		Description: strings.Join(lines, "\n"),
		Footer:      &discordgo.MessageEmbedFooter{Text: time.Now().Format("2006-01-02 15:04:05")},
	}
}

func buildPlayerUpgradeProgressEmbed(player Player) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title: fmt.Sprintf("%s Upgrade Progress", player.Name),
		Fields: []*discordgo.MessageEmbedField{
			makeProgressField("Troops", player.Troops),
			makeProgressField("Heroes", player.Heroes),
			makeProgressField("Spells", player.Spells),
		},
		Footer: &discordgo.MessageEmbedFooter{Text: time.Now().Format("2006-01-02 15:04:05")},
	}
}

func userHasAccount(discordUserID, tag string) bool {
	tag = normalizeTag(tag)
	for _, accountTag := range listUserAccounts(discordUserID) {
		if normalizeTag(accountTag) == tag {
			return true
		}
	}
	return false
}

var helpUsageByCommand = map[string]string{
	"clan overview":             "/clan overview [clan]",
	"help":                      "/help [command]",
	"player equipment-progress": "/player equipment-progress [player]",
	"player overview":           "/player overview [player]",
	"player upgrade-progress":   "/player upgrade-progress [player]",
	"verify add":                "/verify add player",
	"verify list":               "/verify list",
	"verify remove":             "/verify remove player",
	"verify set-main":           "/verify set-main player",
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
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: name, Value: name})
	}
	return choices
}

func handleHelpAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_, focusedValue := getFocusedAutocompleteValue(i)
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: getHelpCommandChoices(focusedValue),
		},
	})
}

func handlePlayerAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_, focusedValue := getFocusedAutocompleteValue(i)
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: playerAutocompleteChoices(interactionUserID(i), focusedValue),
		},
	})
}

func handleClanAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_, focusedValue := getFocusedAutocompleteValue(i)
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: clanAutocompleteChoices(focusedValue),
		},
	})
}

func handleVerifyAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	subcommand, focusedValue := getFocusedAutocompleteValue(i)
	if subcommand == "add" {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{Choices: []*discordgo.ApplicationCommandOptionChoice{}},
		})
		return
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: playerAutocompleteChoices(interactionUserID(i), focusedValue),
		},
	})
}

var CommandAutocompleteHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"clan":   handleClanAutocomplete,
	"help":   handleHelpAutocomplete,
	"player": handlePlayerAutocomplete,
	"verify": handleVerifyAutocomplete,
}

var (
	Commands = []*discordgo.ApplicationCommand{
		{
			Name:        "clan",
			Description: "Clan stats and insights.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "overview",
					Description: "Shows a high-level clan summary.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:         discordgo.ApplicationCommandOptionString,
							Name:         "clan",
							Description:  "Clan tag or known clan name.",
							Required:     false,
							Autocomplete: true,
						},
					},
				},
			},
		},
		{
			Name:        "help",
			Description: "Displays help for commands.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "command",
					Description:  "Command to show usage for.",
					Required:     false,
					Autocomplete: true,
				},
			},
		},
		{
			Name:        "player",
			Description: "Player stats and progression.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "overview",
					Description: "Shows a player summary.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:         discordgo.ApplicationCommandOptionString,
							Name:         "player",
							Description:  "Player tag or known player name.",
							Required:     false,
							Autocomplete: true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "equipment-progress",
					Description: "Shows hero equipment progress.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:         discordgo.ApplicationCommandOptionString,
							Name:         "player",
							Description:  "Player tag or known player name.",
							Required:     false,
							Autocomplete: true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "upgrade-progress",
					Description: "Shows troops, heroes, and spells progression.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:         discordgo.ApplicationCommandOptionString,
							Name:         "player",
							Description:  "Player tag or known player name.",
							Required:     false,
							Autocomplete: true,
						},
					},
				},
			},
		},
		{
			Name:        "verify",
			Description: "Manage linked player accounts.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Link a player tag to your Discord user.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:         discordgo.ApplicationCommandOptionString,
							Name:         "player",
							Description:  "Player tag to link.",
							Required:     true,
							Autocomplete: true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List your linked player accounts.",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Unlink one of your player accounts.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:         discordgo.ApplicationCommandOptionString,
							Name:         "player",
							Description:  "Linked player tag or name.",
							Required:     true,
							Autocomplete: true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "set-main",
					Description: "Set your default player account.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:         discordgo.ApplicationCommandOptionString,
							Name:         "player",
							Description:  "Linked player tag or name.",
							Required:     true,
							Autocomplete: true,
						},
					},
				},
			},
		},
	}

	CommandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"help": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			ctx := getCommandContext(i)
			commandName := strings.ToLower(strings.TrimSpace(stringOption(ctx.options, "command")))

			embed := &discordgo.MessageEmbed{
				Title:  "Available Commands",
				Footer: &discordgo.MessageEmbedFooter{Text: time.Now().Format("2006-01-02 15:04:05")},
			}
			if commandName != "" {
				usage, found := helpUsageByCommand[commandName]
				if !found {
					ephemeralText(s, i, "Unknown command. Use autocomplete for valid commands.")
					return
				}
				embed.Title = "Help: " + commandName
				embed.Description = usage
				respondWithEmbed(s, i, embed)
				return
			}

			fields := make([]*discordgo.MessageEmbedField, 0, len(helpUsageByCommand))
			for _, name := range getHelpCommandNames() {
				fields = append(fields, &discordgo.MessageEmbedField{Name: name, Value: helpUsageByCommand[name]})
			}
			embed.Fields = fields
			respondWithEmbed(s, i, embed)
		},
		"clan": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			ctx := getCommandContext(i)
			if ctx.subcommand != "overview" {
				ephemeralText(s, i, "Unsupported clan subcommand.")
				return
			}

			userID := interactionUserID(i)
			clanTag, ok := resolveClanTag(userID, stringOption(ctx.options, "clan"))
			if !ok || clanTag == "" {
				ephemeralText(s, i, "Provide a clan tag/name or set a main account first.")
				return
			}

			result := getClanByTag(clanTag)
			if !result.OK {
				ephemeralText(s, i, "Failed to fetch clan data: "+result.Error)
				return
			}
			upsertKnownClan(result.Data)
			recordCommandUsage(userID, "clan", result.Data.Tag)
			respondWithEmbed(s, i, buildClanOverviewEmbed(result.Data))
		},
		"player": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			ctx := getCommandContext(i)
			userID := interactionUserID(i)

			playerTag, ok := resolvePlayerTag(userID, stringOption(ctx.options, "player"))
			if !ok || playerTag == "" {
				ephemeralText(s, i, "Provide a player tag/name or set a main account with /verify set-main.")
				return
			}

			result := getPlayerByTag(playerTag)
			if !result.OK {
				ephemeralText(s, i, "Failed to fetch player data: "+result.Error)
				return
			}

			player := result.Data
			upsertKnownPlayer(player)
			if player.Player.Tag != "" && player.Player.Name != "" {
				upsertKnownClan(Clan{Tag: player.Player.Tag, Name: player.Player.Name})
			}
			recordCommandUsage(userID, "player", player.Tag)

			switch ctx.subcommand {
			case "overview":
				respondWithEmbed(s, i, buildPlayerOverviewEmbed(player))
			case "equipment-progress":
				respondWithEmbed(s, i, buildPlayerEquipmentProgressEmbed(player))
			case "upgrade-progress":
				respondWithEmbed(s, i, buildPlayerUpgradeProgressEmbed(player))
			default:
				ephemeralText(s, i, "Unsupported player subcommand.")
			}
		},
		"verify": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			ctx := getCommandContext(i)
			userID := interactionUserID(i)

			switch ctx.subcommand {
			case "add":
				playerTag, ok := resolvePlayerTag(userID, stringOption(ctx.options, "player"))
				if !ok || playerTag == "" {
					ephemeralText(s, i, "Provide a valid player tag.")
					return
				}
				result := getPlayerByTag(playerTag)
				if !result.OK {
					ephemeralText(s, i, "Could not verify that player tag: "+result.Error)
					return
				}
				upsertKnownPlayer(result.Data)
				if result.Data.Player.Tag != "" && result.Data.Player.Name != "" {
					upsertKnownClan(Clan{Tag: result.Data.Player.Tag, Name: result.Data.Player.Name})
				}
				if err := linkUserAccount(userID, result.Data.Tag); err != nil {
					ephemeralText(s, i, "Failed to link account.")
					return
				}
				if _, hasMain := getUserMainAccount(userID); !hasMain {
					_ = setMainUserAccount(userID, result.Data.Tag)
				}
				ephemeralText(s, i, "Linked "+result.Data.Name+" ("+result.Data.Tag+").")
			case "list":
				accounts := listUserAccounts(userID)
				if len(accounts) == 0 {
					ephemeralText(s, i, "No linked accounts yet. Use /verify add.")
					return
				}
				mainTag, _ := getUserMainAccount(userID)
				lines := make([]string, 0, len(accounts))
				for _, tag := range accounts {
					label := tag
					if normalizeTag(tag) == normalizeTag(mainTag) {
						label += " (main)"
					}
					lines = append(lines, "- "+label)
				}
				ephemeralText(s, i, "Linked accounts:\n"+strings.Join(lines, "\n"))
			case "remove":
				playerTag, ok := resolvePlayerTag(userID, stringOption(ctx.options, "player"))
				if !ok || playerTag == "" {
					ephemeralText(s, i, "Provide a linked account to remove.")
					return
				}
				if !userHasAccount(userID, playerTag) {
					ephemeralText(s, i, "That player is not linked to your account.")
					return
				}
				_ = removeUserAccount(userID, playerTag)
				ephemeralText(s, i, "Removed linked account "+playerTag+".")
			case "set-main":
				playerTag, ok := resolvePlayerTag(userID, stringOption(ctx.options, "player"))
				if !ok || playerTag == "" {
					ephemeralText(s, i, "Provide one of your linked accounts.")
					return
				}
				if !userHasAccount(userID, playerTag) {
					ephemeralText(s, i, "That player is not linked to your account.")
					return
				}
				_ = setMainUserAccount(userID, playerTag)
				ephemeralText(s, i, "Set "+playerTag+" as your main account.")
			default:
				ephemeralText(s, i, "Unsupported verify subcommand.")
			}
		},
	}
)
