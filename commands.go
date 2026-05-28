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

const (
	verifyTokenModalPrefix = "verify:add:"
	verifyTokenInputID     = "token"
)

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

func modalTextInputValue(i *discordgo.InteractionCreate, inputID string) string {
	data := i.ModalSubmitData()
	for _, component := range data.Components {
		row, ok := component.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, rowComponent := range row.Components {
			input, ok := rowComponent.(*discordgo.TextInput)
			if !ok {
				continue
			}
			if input.CustomID == inputID {
				return strings.TrimSpace(input.Value)
			}
		}
	}
	return ""
}

func handleVerifyTokenModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	if !strings.HasPrefix(customID, verifyTokenModalPrefix) {
		ephemeralText(s, i, "Unsupported modal submission.")
		return
	}

	playerTag := normalizeTag(strings.TrimPrefix(customID, verifyTokenModalPrefix))
	if playerTag == "" {
		ephemeralText(s, i, "Missing player tag for verification.")
		return
	}

	token := modalTextInputValue(i, verifyTokenInputID)
	if token == "" {
		ephemeralText(s, i, "Enter the in-game API token from Clash settings.")
		return
	}

	verifyResult := verifyPlayerToken(playerTag, token)
	if !verifyResult.OK {
		ephemeralText(s, i, "Verification failed: "+verifyResult.Error)
		return
	}
	if !strings.EqualFold(verifyResult.Data.Status, "ok") {
		ephemeralText(s, i, "Verification failed: invalid player tag or token.")
		return
	}

	playerResult := getPlayerByTag(playerTag)
	if !playerResult.OK {
		ephemeralText(s, i, "Player verified but profile fetch failed: "+playerResult.Error)
		return
	}

	userID := interactionUserID(i)
	upsertKnownPlayer(playerResult.Data)
	if playerResult.Data.Player.Tag != "" && playerResult.Data.Player.Name != "" {
		upsertKnownClan(Clan{Tag: playerResult.Data.Player.Tag, Name: playerResult.Data.Player.Name})
	}
	if err := linkUserAccount(userID, playerResult.Data.Tag); err != nil {
		ephemeralText(s, i, "Failed to link account.")
		return
	}
	if _, hasMain := getUserMainAccount(userID); !hasMain {
		_ = setMainUserAccount(userID, playerResult.Data.Tag)
	}

	respondWithEphemeralEmbed(
		s,
		i,
		buildStatusEmbed(
			"Account Verified",
			"Verified and linked "+playerResult.Data.Name+" ("+playerResult.Data.Tag+").",
		),
	)
}

func respondWithEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func respondWithEphemeralEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

func buildStatusEmbed(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Footer:      &discordgo.MessageEmbedFooter{Text: time.Now().Format("2006-01-02 15:04:05")},
	}
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
	labels := make([]string, 0, len(clan.Labels))
	for _, label := range clan.Labels {
		if label.Name != "" {
			labels = append(labels, label.Name)
		}
	}
	if len(labels) == 0 {
		labels = append(labels, "None")
	}

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
			{Name: "Required TH", Value: fmt.Sprintf("%d", clan.RequiredTownhallLevel), Inline: true},
			{Name: "Required Trophies", Value: fmt.Sprintf("%d", clan.RequiredTrophies), Inline: true},
			{Name: "Location", Value: clan.Location.Name, Inline: true},
			{Name: "Type", Value: string(clan.Type), Inline: true},
			{Name: "War Win Streak", Value: fmt.Sprintf("%d", clan.WarWinStreak), Inline: true},
			{Name: "Labels", Value: strings.Join(labels, ", ")},
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
			{Name: "TH Weapon", Value: fmt.Sprintf("%d", player.TownHallWeaponLevel), Inline: true},
			{Name: "Exp Level", Value: fmt.Sprintf("%d", player.ExpLevel), Inline: true},
			{Name: "Trophies", Value: fmt.Sprintf("%d", player.Trophies), Inline: true},
			{Name: "Best Trophies", Value: fmt.Sprintf("%d", player.BestTrophies), Inline: true},
			{Name: "War Stars", Value: fmt.Sprintf("%d", player.WarStars), Inline: true},
			{Name: "Clan", Value: clanName, Inline: true},
			{Name: "Role", Value: string(player.Role), Inline: true},
			{Name: "War Preference", Value: string(player.WarPreference), Inline: true},
			{Name: "Attack Wins", Value: fmt.Sprintf("%d", player.AttackWins), Inline: true},
			{Name: "Defense Wins", Value: fmt.Sprintf("%d", player.DefenseWins), Inline: true},
			{Name: "Donations", Value: fmt.Sprintf("%d", player.Donations), Inline: true},
			{Name: "Received", Value: fmt.Sprintf("%d", player.DonationsReceived), Inline: true},
			{Name: "Builder Hall", Value: fmt.Sprintf("%d", player.BuilderHallLevel), Inline: true},
			{Name: "Builder Trophies", Value: fmt.Sprintf("%d", player.BuilderBaseTrophies), Inline: true},
			{Name: "Best Builder", Value: fmt.Sprintf("%d", player.BestBuilderBaseTrophies), Inline: true},
			{Name: "Capital Contributions", Value: fmt.Sprintf("%d", player.ClanCapitalContributions), Inline: true},
			{Name: "League", Value: player.League.Name, Inline: true},
			{Name: "Builder League", Value: player.BuilderBaseLeague.Name, Inline: true},
			{Name: "Legend Trophies", Value: fmt.Sprintf("%d", player.LegendStatistics.LegendTrophies), Inline: true},
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
		Title:       fmt.Sprintf("%s Player Equipment", player.Name),
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

func buildPlayerItemsEmbed(player Player, title string, items []PlayerItemLevel, emptyMessage string) *discordgo.MessageEmbed {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		line := fmt.Sprintf("%s %d/%d", item.Name, item.Level, item.MaxLevel)
		if len(item.Equipment) > 0 {
			line += fmt.Sprintf(" (%d equipment)", len(item.Equipment))
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = append(lines, emptyMessage)
	}
	if len(lines) > 20 {
		lines = lines[:20]
	}
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s %s", player.Name, title),
		Description: strings.Join(lines, "\n"),
		Footer:      &discordgo.MessageEmbedFooter{Text: time.Now().Format("2006-01-02 15:04:05")},
	}
}

func buildPlayerAchievementsEmbed(player Player) *discordgo.MessageEmbed {
	lines := make([]string, 0, len(player.Achievements))
	for _, achievement := range player.Achievements {
		lines = append(lines, fmt.Sprintf("%s: %d/%d (%d stars)", achievement.Name, achievement.Value, achievement.Target, achievement.Stars))
	}
	if len(lines) == 0 {
		lines = append(lines, "No achievements found.")
	}
	if len(lines) > 20 {
		lines = lines[:20]
	}
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s Achievements", player.Name),
		Description: strings.Join(lines, "\n"),
		Footer:      &discordgo.MessageEmbedFooter{Text: time.Now().Format("2006-01-02 15:04:05")},
	}
}

func buildVerifyListEmbed(accounts []string, mainTag string) *discordgo.MessageEmbed {
	lines := make([]string, 0, len(accounts))
	for _, tag := range accounts {
		line := tag
		if normalizeTag(tag) == normalizeTag(mainTag) {
			line += " (main)"
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = append(lines, "No linked accounts.")
	}
	return &discordgo.MessageEmbed{
		Title:       "Linked Accounts",
		Description: strings.Join(lines, "\n"),
		Footer:      &discordgo.MessageEmbedFooter{Text: time.Now().Format("2006-01-02 15:04:05")},
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

func resolveAndFetchPlayer(s *discordgo.Session, i *discordgo.InteractionCreate) (Player, bool) {
	ctx := getCommandContext(i)
	userID := interactionUserID(i)

	playerTag, ok := resolvePlayerTag(userID, stringOption(ctx.options, "player"))
	if !ok || playerTag == "" {
		ephemeralText(s, i, "Provide a player tag/name or set a main account with /verify-set-main.")
		return Player{}, false
	}

	result := getPlayerByTag(playerTag)
	if !result.OK {
		ephemeralText(s, i, "Failed to fetch player data: "+result.Error)
		return Player{}, false
	}

	player := result.Data
	upsertKnownPlayer(player)
	if player.Player.Tag != "" && player.Player.Name != "" {
		upsertKnownClan(Clan{Tag: player.Player.Tag, Name: player.Player.Name})
	}
	recordCommandUsage(userID, "player", player.Tag)
	return player, true
}

var helpUsageByCommand = map[string]string{
	"clan":                  "/clan [clan]",
	"help":                  "/help [command]",
	"player":                "/player [player]",
	"player achievements":   "/player-achievements [player]",
	"player equipment":      "/player-equipment [player]",
	"player heroes":         "/player-heroes [player]",
	"player spells":         "/player-spells [player]",
	"player troops":         "/player-troops [player]",
	"player upgrade":        "/player-upgrade-progress [player]",
	"verify":                "/verify player",
	"verify list":           "/verify-list",
	"verify remove":         "/verify-remove player",
	"verify set-main":       "/verify-set-main player",
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
	_, focusedValue := getFocusedAutocompleteValue(i)
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: playerAutocompleteChoices(interactionUserID(i), focusedValue),
		},
	})
}

var CommandAutocompleteHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"clan":                  handleClanAutocomplete,
	"help":                  handleHelpAutocomplete,
	"player":                handlePlayerAutocomplete,
	"player-achievements":   handlePlayerAutocomplete,
	"player-equipment":      handlePlayerAutocomplete,
	"player-heroes":         handlePlayerAutocomplete,
	"player-spells":         handlePlayerAutocomplete,
	"player-troops":         handlePlayerAutocomplete,
	"player-upgrade-progress": handlePlayerAutocomplete,
	"verify":                handleVerifyAutocomplete,
	"verify-remove":         handleVerifyAutocomplete,
	"verify-set-main":       handleVerifyAutocomplete,
}

var (
	Commands = []*discordgo.ApplicationCommand{
		{
			Name:        "clan",
			Description: "Clan stats and insights.",
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
			Description: "Player overview.",
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
			Name:        "player-equipment",
			Description: "Player equipment levels.",
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
			Name:        "player-heroes",
			Description: "Player hero levels.",
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
			Name:        "player-troops",
			Description: "Player troop levels.",
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
			Name:        "player-spells",
			Description: "Player spell levels.",
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
			Name:        "player-achievements",
			Description: "Player achievement progress.",
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
			Name:        "player-upgrade-progress",
			Description: "Troops, heroes, and spells progression.",
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
			Name:        "verify",
			Description: "Verify and link a player account.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "player",
					Description:  "Player tag to verify and link.",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
		{
			Name:        "verify-list",
			Description: "List your linked player accounts.",
		},
		{
			Name:        "verify-remove",
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
			Name:        "verify-set-main",
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
			player, ok := resolveAndFetchPlayer(s, i)
			if !ok {
				return
			}
			respondWithEmbed(s, i, buildPlayerOverviewEmbed(player))
		},
		"player-equipment": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			player, ok := resolveAndFetchPlayer(s, i)
			if !ok {
				return
			}
			respondWithEmbed(s, i, buildPlayerEquipmentProgressEmbed(player))
		},
		"player-heroes": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			player, ok := resolveAndFetchPlayer(s, i)
			if !ok {
				return
			}
			respondWithEmbed(s, i, buildPlayerItemsEmbed(player, "Heroes", player.Heroes, "No heroes found."))
		},
		"player-troops": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			player, ok := resolveAndFetchPlayer(s, i)
			if !ok {
				return
			}
			respondWithEmbed(s, i, buildPlayerItemsEmbed(player, "Troops", player.Troops, "No troops found."))
		},
		"player-spells": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			player, ok := resolveAndFetchPlayer(s, i)
			if !ok {
				return
			}
			respondWithEmbed(s, i, buildPlayerItemsEmbed(player, "Spells", player.Spells, "No spells found."))
		},
		"player-achievements": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			player, ok := resolveAndFetchPlayer(s, i)
			if !ok {
				return
			}
			respondWithEmbed(s, i, buildPlayerAchievementsEmbed(player))
		},
		"player-upgrade-progress": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			player, ok := resolveAndFetchPlayer(s, i)
			if !ok {
				return
			}
			respondWithEmbed(s, i, buildPlayerUpgradeProgressEmbed(player))
		},
		"verify": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			ctx := getCommandContext(i)
			userID := interactionUserID(i)
			playerTag, ok := resolvePlayerTag(userID, stringOption(ctx.options, "player"))
			if !ok || playerTag == "" {
				ephemeralText(s, i, "Provide a valid player tag.")
				return
			}
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseModal,
				Data: &discordgo.InteractionResponseData{
					CustomID: verifyTokenModalPrefix + playerTag,
					Title:    "Verify Player Account",
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.TextInput{
									CustomID:    verifyTokenInputID,
									Label:       "Player API Token",
									Style:       discordgo.TextInputShort,
									Placeholder: "Paste one-time token from in-game settings",
									Required:    true,
									MinLength:   1,
									MaxLength:   64,
								},
							},
						},
					},
				},
			})
		},
		"verify-list": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			userID := interactionUserID(i)
			accounts := listUserAccounts(userID)
			mainTag, _ := getUserMainAccount(userID)
			if len(accounts) == 0 {
				respondWithEphemeralEmbed(s, i, buildStatusEmbed("Linked Accounts", "No linked accounts yet. Use `/verify player:<tag>` to start."))
				return
			}
			respondWithEphemeralEmbed(s, i, buildVerifyListEmbed(accounts, mainTag))
		},
		"verify-remove": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			ctx := getCommandContext(i)
			userID := interactionUserID(i)
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
			respondWithEphemeralEmbed(s, i, buildStatusEmbed("Account Removed", "Removed linked account "+playerTag+"."))
		},
		"verify-set-main": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			ctx := getCommandContext(i)
			userID := interactionUserID(i)
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
			respondWithEphemeralEmbed(s, i, buildStatusEmbed("Main Account Updated", "Set "+playerTag+" as your main account."))
		},
	}
)
