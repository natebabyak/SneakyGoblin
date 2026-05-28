package main

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type commandContext struct {
	subcommand string
	options    map[string]*discordgo.ApplicationCommandInteractionDataOption
}

const (
	verifyTokenModalPrefix = "verify:verify:"
	verifyTokenInputID     = "token"
	botWatermark           = "SneakyGoblin"
	embedColor             = 0x00C950
	clanTabPrefix          = "clan-tab:"
	clanTabOverview        = "overview"
	clanTabMembers         = "members"
	clanTabWars            = "wars"
	clanTabCapital         = "capital"
	clanMemPrefix          = "clan-mem:"
	clanMemSortPrefix      = "clan-mem-sort:"
	clanWarPrefix          = "clan-war:"
	clanWarSortPrefix      = "clan-war-sort:"
	clanMembersPerPage     = 15
	clanWarsPerPage        = 15
	clanMemberDefaultSort  = "league-trophies"
	clanWarDefaultSort        = "date"
	playerAchPrefix           = "player-ach:"
	playerAchSortPrefix       = "player-ach-sort:"
	playerAchievementsPerPage = 15
	playerAchDefaultSort      = "default"
)

type clanPanelState struct {
	memPage, memTotalPages int
	memSort                string
	warPage, warTotalPages int
	warSort                string
}

func defaultClanPanelState() clanPanelState {
	return clanPanelState{
		memSort:       clanMemberDefaultSort,
		warSort:       clanWarDefaultSort,
		memTotalPages: 1,
		warTotalPages: 1,
	}
}

var botAvatarURL string

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
			playerResult.Data.Name,
			embedDescriptionWithTag(playerResult.Data.Tag, "Verified and linked to your Discord account."),
			playerThumbnailURL(playerResult.Data),
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

func statsEmbedFooter() *discordgo.MessageEmbedFooter {
	footer := &discordgo.MessageEmbedFooter{
		Text: botWatermark + " · " + time.Now().Format("2006-01-02 15:04:05"),
	}
	if botAvatarURL != "" {
		footer.IconURL = botAvatarURL
	}
	return footer
}

func withStatsEmbed(embed *discordgo.MessageEmbed, thumbnailURL string) *discordgo.MessageEmbed {
	embed.Color = embedColor
	embed.Footer = statsEmbedFooter()
	if thumbnailURL != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: thumbnailURL}
	}
	return embed
}

func tagSubheading(tag string) string {
	return "-# " + normalizeTag(tag)
}

func formatCompactNumber(n int) string {
	abs := n
	sign := ""
	if n < 0 {
		sign = "-"
		abs = -n
	}
	if abs < 1000 {
		return sign + strconv.Itoa(n)
	}

	type scale struct {
		div    float64
		suffix string
	}
	for _, s := range []scale{{1e9, "B"}, {1e6, "M"}, {1e3, "K"}} {
		if float64(abs) >= s.div {
			return sign + formatWithSigFigs(float64(abs)/s.div, 3) + s.suffix
		}
	}
	return sign + strconv.Itoa(n)
}

func formatWithSigFigs(v float64, sig int) string {
	if v == 0 {
		return "0"
	}
	magnitude := int(math.Floor(math.Log10(v)))
	decimals := sig - 1 - magnitude
	if decimals < 0 {
		decimals = 0
	}
	return strconv.FormatFloat(v, 'f', decimals, 64)
}

func embedDescriptionWithTag(tag, body string) string {
	tagLine := tagSubheading(tag)
	body = strings.TrimSpace(body)
	if body == "" {
		return tagLine
	}
	return tagLine + "\n\n" + body
}

func clanBadgeURL(clan Clan) string {
	if clan.BadgeUrls.Medium != "" {
		return clan.BadgeUrls.Medium
	}
	if clan.BadgeUrls.Large != "" {
		return clan.BadgeUrls.Large
	}
	return clan.BadgeUrls.Small
}

func playerThumbnailURL(player Player) string {
	if player.LeagueTier.IconUrls.Large != "" {
		return player.LeagueTier.IconUrls.Large
	}
	if player.LeagueTier.IconUrls.Small != "" {
		return player.LeagueTier.IconUrls.Small
	}
	if player.League.IconUrls.Small != "" {
		return player.League.IconUrls.Small
	}
	if player.League.IconUrls.Tiny != "" {
		return player.League.IconUrls.Tiny
	}
	if player.Player.BadgeUrls.Medium != "" {
		return player.Player.BadgeUrls.Medium
	}
	return player.Player.BadgeUrls.Large
}

func buildStatusEmbed(title, description, thumbnailURL string) *discordgo.MessageEmbed {
	return withStatsEmbed(&discordgo.MessageEmbed{
		Title:       title,
		Description: description,
	}, thumbnailURL)
}

func playerSubcommand(name, description string) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionSubCommand,
		Name:        name,
		Description: description,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "player",
				Description:  "Player tag or known player name.",
				Required:     false,
				Autocomplete: true,
			},
		},
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

func clanTagFromPlayer(player Player) (string, bool) {
	clanTag := strings.TrimSpace(player.Player.Tag)
	if clanTag == "" {
		return "", false
	}
	if strings.EqualFold(string(player.Role), "notMember") {
		return "", false
	}
	return normalizeTag(clanTag), true
}

func resolveClanTagFromVerifiedAccounts(discordUserID string) (string, bool) {
	accounts := listUserAccounts(discordUserID)
	if len(accounts) == 0 {
		return "", false
	}

	ordered := make([]string, 0, len(accounts))
	if mainTag, ok := getUserMainAccount(discordUserID); ok {
		ordered = append(ordered, mainTag)
		for _, tag := range accounts {
			if normalizeTag(tag) != normalizeTag(mainTag) {
				ordered = append(ordered, tag)
			}
		}
	} else {
		ordered = append(ordered, accounts...)
	}

	for _, playerTag := range ordered {
		if clanTag, ok := getKnownClanTagForPlayer(playerTag); ok {
			return clanTag, true
		}

		result := getPlayerByTag(playerTag)
		if !result.OK {
			continue
		}
		if clanTag, ok := clanTagFromPlayer(result.Data); ok {
			upsertKnownPlayer(result.Data)
			return clanTag, true
		}
	}
	return "", false
}

func resolveClanTag(discordUserID, input string) (string, bool) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return resolveClanTagFromVerifiedAccounts(discordUserID)
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

	return withStatsEmbed(&discordgo.MessageEmbed{
		Title:       clan.Name,
		Description: embedDescriptionWithTag(clan.Tag, clan.Description),
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
			{Name: "War League", Value: clan.WarLeague.Name, Inline: true},
			{Name: "Capital League", Value: clan.CapitalLeague.Name, Inline: true},
			{Name: "Builder Base", Value: fmt.Sprintf("%d", clan.ClanBuilderBasePoints), Inline: true},
			{Name: "War Log Public", Value: boolString(clan.IsWarLogPublic), Inline: true},
			{Name: "Family Friendly", Value: boolString(clan.IsFamilyFriendly), Inline: true},
			{Name: "Chat Language", Value: clan.ChatLanguage.Name, Inline: true},
			{Name: "Labels", Value: strings.Join(labels, ", ")},
		},
	}, clanBadgeURL(clan))
}

func boolString(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

func clanTabButtonID(tab, tag string) string {
	return clanTabPrefix + tab + ":" + strings.TrimPrefix(normalizeTag(tag), "#")
}

func parseClanTabButtonID(customID string) (tab, tag string, ok bool) {
	if !strings.HasPrefix(customID, clanTabPrefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(customID, clanTabPrefix)
	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], normalizeTag(parts[1]), true
}

var clanMemberSorts = []struct {
	key   string
	label string
}{
	{key: "league-trophies", label: "League & Trophies"},
	{key: "trophies", label: "Trophies"},
	{key: "th", label: "Town Hall"},
	{key: "role", label: "Role"},
	{key: "donations", label: "Troops Donated"},
	{key: "received", label: "Troops Received"},
	{key: "exp", label: "XP Level"},
	{key: "bb-trophies", label: "Builder Trophies"},
}

func normalizeClanMemberSort(sort string) string {
	if sort == "league" {
		return clanMemberDefaultSort
	}
	if isValidClanMemberSort(sort) {
		return sort
	}
	return clanMemberDefaultSort
}

func clanTabComponents(clanTag, activeTab string) []discordgo.MessageComponent {
	tabs := []struct {
		id    string
		label string
	}{
		{clanTabOverview, "Overview"},
		{clanTabMembers, "Members"},
		{clanTabWars, "Wars"},
		{clanTabCapital, "Clan Capital"},
	}
	buttons := make([]discordgo.MessageComponent, 0, len(tabs))
	for _, tab := range tabs {
		buttons = append(buttons, discordgo.Button{
			Label:    tab.label,
			Style:    discordgo.SecondaryButton,
			CustomID: clanTabButtonID(tab.id, clanTag),
			Disabled: tab.id == activeTab,
		})
	}
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: buttons}}
}

func clanMemButtonID(action, tag, sort string, page int) string {
	return fmt.Sprintf("%s%s:%s:%d:%s", clanMemPrefix, action, strings.TrimPrefix(normalizeTag(tag), "#"), page, sort)
}

func parseClanMemButtonID(customID string) (action, tag, sort string, page int, ok bool) {
	if !strings.HasPrefix(customID, clanMemPrefix) {
		return "", "", "", 0, false
	}
	parts := strings.Split(strings.TrimPrefix(customID, clanMemPrefix), ":")
	if len(parts) != 4 {
		return "", "", "", 0, false
	}
	action = parts[0]
	if action != "p" && action != "n" {
		return "", "", "", 0, false
	}
	page, err := strconv.Atoi(parts[2])
	if err != nil || page < 0 {
		return "", "", "", 0, false
	}
	sort = normalizeClanMemberSort(parts[3])
	return action, normalizeTag(parts[1]), sort, page, true
}

func isValidClanMemberSort(sort string) bool {
	for _, option := range clanMemberSorts {
		if option.key == sort {
			return true
		}
	}
	return false
}

func clanMemberSortLabel(sort string) string {
	for _, option := range clanMemberSorts {
		if option.key == sort {
			return option.label
		}
	}
	return "League & Trophies"
}

func memberLeagueSortKey(member ClanMember) int {
	if member.LeagueTier.Id > 0 {
		return member.LeagueTier.Id
	}
	return member.League.Id
}

func memberLeagueName(member ClanMember) string {
	if member.LeagueTier.Name != "" {
		return member.LeagueTier.Name
	}
	if member.League.Name != "" {
		return member.League.Name
	}
	return "Unranked"
}

func stripLeagueWordFromName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " League ", " ")
	name = strings.ReplaceAll(name, " league ", " ")
	name = strings.TrimSuffix(name, " League")
	name = strings.TrimSuffix(name, " league")
	return strings.TrimSpace(name)
}

func roleSortRank(role Role) int {
	switch strings.ToLower(string(role)) {
	case "leader":
		return 0
	case "coleader":
		return 1
	case "admin", "elder":
		return 2
	case "member":
		return 3
	default:
		return 4
	}
}

func sortClanMembers(members []ClanMember, sortKey string) {
	sortKey = normalizeClanMemberSort(sortKey)
	tieTrophies := func(i, j int) bool {
		if members[i].Trophies != members[j].Trophies {
			return members[i].Trophies > members[j].Trophies
		}
		return members[i].ClanRank < members[j].ClanRank
	}

	switch sortKey {
	case "trophies":
		sort.Slice(members, func(i, j int) bool {
			if members[i].Trophies != members[j].Trophies {
				return members[i].Trophies > members[j].Trophies
			}
			if memberLeagueSortKey(members[i]) != memberLeagueSortKey(members[j]) {
				return memberLeagueSortKey(members[i]) > memberLeagueSortKey(members[j])
			}
			return members[i].ClanRank < members[j].ClanRank
		})
	case "th":
		sort.Slice(members, func(i, j int) bool {
			if members[i].TownHallLevel != members[j].TownHallLevel {
				return members[i].TownHallLevel > members[j].TownHallLevel
			}
			return tieTrophies(i, j)
		})
	case "role":
		sort.Slice(members, func(i, j int) bool {
			ri, rj := roleSortRank(members[i].Role), roleSortRank(members[j].Role)
			if ri != rj {
				return ri < rj
			}
			return tieTrophies(i, j)
		})
	case "donations":
		sort.Slice(members, func(i, j int) bool {
			if members[i].Donations != members[j].Donations {
				return members[i].Donations > members[j].Donations
			}
			return tieTrophies(i, j)
		})
	case "received":
		sort.Slice(members, func(i, j int) bool {
			if members[i].DonationsReceived != members[j].DonationsReceived {
				return members[i].DonationsReceived > members[j].DonationsReceived
			}
			return tieTrophies(i, j)
		})
	case "exp":
		sort.Slice(members, func(i, j int) bool {
			if members[i].ExpLevel != members[j].ExpLevel {
				return members[i].ExpLevel > members[j].ExpLevel
			}
			return tieTrophies(i, j)
		})
	case "bb-trophies":
		sort.Slice(members, func(i, j int) bool {
			if members[i].BuilderBaseTrophies != members[j].BuilderBaseTrophies {
				return members[i].BuilderBaseTrophies > members[j].BuilderBaseTrophies
			}
			return tieTrophies(i, j)
		})
	default:
		sort.Slice(members, func(i, j int) bool {
			li, lj := memberLeagueSortKey(members[i]), memberLeagueSortKey(members[j])
			if li != lj {
				return li > lj
			}
			if members[i].Trophies != members[j].Trophies {
				return members[i].Trophies > members[j].Trophies
			}
			return members[i].ClanRank < members[j].ClanRank
		})
	}
}

func formatClanMemberSortMetric(member ClanMember, sortKey string) string {
	switch normalizeClanMemberSort(sortKey) {
	case "trophies":
		return fmt.Sprintf("`%d`", member.Trophies)
	case "th":
		return fmt.Sprintf("`%d`", member.TownHallLevel)
	case "role":
		return formatClanMemberRole(member.Role)
	case "donations":
		return fmt.Sprintf("`%d`", member.Donations)
	case "received":
		return fmt.Sprintf("`%d`", member.DonationsReceived)
	case "exp":
		return fmt.Sprintf("`%d`", member.ExpLevel)
	case "bb-trophies":
		return fmt.Sprintf("`%d`", member.BuilderBaseTrophies)
	default:
		return fmt.Sprintf("`%d`", member.Trophies)
	}
}

func formatClanMemberLeagueTrophiesCell(member ClanMember) string {
	return fmt.Sprintf("`%s` & `%d`", stripLeagueWordFromName(memberLeagueName(member)), member.Trophies)
}

func formatClanMemberIndexName(index int, name string) string {
	return fmt.Sprintf("**%d.** %s", index, name)
}

func formatClanMemberTagCell(member ClanMember) string {
	tag := normalizeTag(member.Tag)
	if tag == "" {
		tag = "#UNKNOWN"
	}
	return "`" + tag + "`"
}

func clanMemberColumnField(name string, lines []string) *discordgo.MessageEmbedField {
	return &discordgo.MessageEmbedField{
		Name:   name,
		Value:  strings.Join(lines, "\n"),
		Inline: true,
	}
}

func clanMemberTableFields(pageMembers []ClanMember, sortKey string, startIndex int) []*discordgo.MessageEmbedField {
	if len(pageMembers) == 0 {
		return nil
	}

	sortKey = normalizeClanMemberSort(sortKey)
	nameLines := make([]string, 0, len(pageMembers))
	tagLines := make([]string, 0, len(pageMembers))
	for i, member := range pageMembers {
		idx := startIndex + i + 1
		nameLines = append(nameLines, formatClanMemberIndexName(idx, member.Name))
		tagLines = append(tagLines, formatClanMemberTagCell(member))
	}

	fields := []*discordgo.MessageEmbedField{
		clanMemberColumnField("Clan Member", nameLines),
		clanMemberColumnField("Tag", tagLines),
	}

	if sortKey == "league-trophies" {
		rankLines := make([]string, 0, len(pageMembers))
		for _, member := range pageMembers {
			rankLines = append(rankLines, formatClanMemberLeagueTrophiesCell(member))
		}
		fields = append(fields, clanMemberColumnField(clanMemberSortLabel(sortKey), rankLines))
		return fields
	}

	metricLines := make([]string, 0, len(pageMembers))
	for _, member := range pageMembers {
		metricLines = append(metricLines, formatClanMemberSortMetric(member, sortKey))
	}
	fields = append(fields, clanMemberColumnField(clanMemberSortLabel(sortKey), metricLines))
	return fields
}

var clanWarSorts = []struct {
	key   string
	label string
}{
	{key: "date", label: "Date"},
	{key: "result", label: "Result"},
	{key: "opponent", label: "Opponent"},
	{key: "stars", label: "Stars"},
	{key: "destruction", label: "Destruction"},
	{key: "size", label: "War Size"},
}

func normalizeClanWarSort(sort string) string {
	if isValidClanWarSort(sort) {
		return sort
	}
	return clanWarDefaultSort
}

func isValidClanWarSort(sort string) bool {
	for _, option := range clanWarSorts {
		if option.key == sort {
			return true
		}
	}
	return false
}

func clanWarSortLabel(sort string) string {
	for _, option := range clanWarSorts {
		if option.key == sort {
			return option.label
		}
	}
	return "Date"
}

func parseWarEndTime(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	for _, layout := range []string{"20060102T150405.000Z", "20060102T150405Z", time.RFC3339} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t
		}
	}
	return time.Time{}
}

func formatWarEndTime(raw string) string {
	t := parseWarEndTime(raw)
	if t.IsZero() {
		return raw
	}
	return t.Format("Jan 2, 2006")
}

func warResultSortRank(result string) int {
	switch strings.ToLower(strings.TrimSpace(result)) {
	case "win":
		return 0
	case "tie":
		return 1
	case "lose", "loss":
		return 2
	default:
		return 3
	}
}

func sortClanWarLog(entries []warLogEntry, sortKey string) {
	sortKey = normalizeClanWarSort(sortKey)
	tieDate := func(i, j int) bool {
		ti, tj := parseWarEndTime(entries[i].EndTime), parseWarEndTime(entries[j].EndTime)
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return entries[i].Opponent.Name < entries[j].Opponent.Name
	}

	switch sortKey {
	case "result":
		sort.Slice(entries, func(i, j int) bool {
			ri, rj := warResultSortRank(entries[i].Result), warResultSortRank(entries[j].Result)
			if ri != rj {
				return ri < rj
			}
			return tieDate(i, j)
		})
	case "opponent":
		sort.Slice(entries, func(i, j int) bool {
			ni, nj := strings.ToLower(entries[i].Opponent.Name), strings.ToLower(entries[j].Opponent.Name)
			if ni != nj {
				return ni < nj
			}
			return tieDate(i, j)
		})
	case "stars":
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Clan.Stars != entries[j].Clan.Stars {
				return entries[i].Clan.Stars > entries[j].Clan.Stars
			}
			return tieDate(i, j)
		})
	case "destruction":
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Clan.DestructionPercentage != entries[j].Clan.DestructionPercentage {
				return entries[i].Clan.DestructionPercentage > entries[j].Clan.DestructionPercentage
			}
			return tieDate(i, j)
		})
	case "size":
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].TeamSize != entries[j].TeamSize {
				return entries[i].TeamSize > entries[j].TeamSize
			}
			return tieDate(i, j)
		})
	default:
		sort.Slice(entries, func(i, j int) bool {
			return tieDate(i, j)
		})
	}
}

func formatWarIndexResult(index int, entry warLogEntry) string {
	label := strings.ToUpper(strings.TrimSpace(entry.Result))
	if label == "" {
		label = "—"
	}
	return fmt.Sprintf("**%d.** %s · `%d`v`%d`", index, label, entry.TeamSize, entry.TeamSize)
}

func formatWarMatchCell(entry warLogEntry) string {
	return fmt.Sprintf(
		"**%s**\n`%d` - `%d`⭐ · `%.0f`%% - `%.0f`%%",
		entry.Opponent.Name,
		entry.Clan.Stars, entry.Opponent.Stars,
		entry.Clan.DestructionPercentage, entry.Opponent.DestructionPercentage,
	)
}

func warLogOutcomeCounts(entries []warLogEntry) (wins, losses, ties int) {
	for _, entry := range entries {
		switch strings.ToLower(strings.TrimSpace(entry.Result)) {
		case "win":
			wins++
		case "tie":
			ties++
		case "lose", "loss":
			losses++
		}
	}
	return wins, losses, ties
}

func clanWarTableFields(pageEntries []warLogEntry, startIndex int) []*discordgo.MessageEmbedField {
	if len(pageEntries) == 0 {
		return nil
	}

	resultLines := make([]string, 0, len(pageEntries))
	matchLines := make([]string, 0, len(pageEntries))
	endLines := make([]string, 0, len(pageEntries))
	for i, entry := range pageEntries {
		idx := startIndex + i + 1
		resultLines = append(resultLines, formatWarIndexResult(idx, entry))
		matchLines = append(matchLines, formatWarMatchCell(entry))
		endLines = append(endLines, formatWarEndTime(entry.EndTime))
	}

	return []*discordgo.MessageEmbedField{
		clanMemberColumnField("# · Result", resultLines),
		clanMemberColumnField("Matchup", matchLines),
		clanMemberColumnField("Ended", endLines),
	}
}

func clanWarButtonID(action, tag, sort string, page int) string {
	return fmt.Sprintf("%s%s:%s:%d:%s", clanWarPrefix, action, strings.TrimPrefix(normalizeTag(tag), "#"), page, sort)
}

func parseClanWarButtonID(customID string) (action, tag, sort string, page int, ok bool) {
	if !strings.HasPrefix(customID, clanWarPrefix) {
		return "", "", "", 0, false
	}
	parts := strings.Split(strings.TrimPrefix(customID, clanWarPrefix), ":")
	if len(parts) != 4 {
		return "", "", "", 0, false
	}
	action = parts[0]
	if action != "p" && action != "n" {
		return "", "", "", 0, false
	}
	page, err := strconv.Atoi(parts[2])
	if err != nil || page < 0 {
		return "", "", "", 0, false
	}
	sort = normalizeClanWarSort(parts[3])
	return action, normalizeTag(parts[1]), sort, page, true
}

func clanWarSortSelectID(tag string, page int) string {
	return fmt.Sprintf("%s%s:%d", clanWarSortPrefix, strings.TrimPrefix(normalizeTag(tag), "#"), page)
}

func parseClanWarSortSelectID(customID string) (tag string, page int, ok bool) {
	if !strings.HasPrefix(customID, clanWarSortPrefix) {
		return "", 0, false
	}
	parts := strings.Split(strings.TrimPrefix(customID, clanWarSortPrefix), ":")
	if len(parts) != 2 {
		return "", 0, false
	}
	page, err := strconv.Atoi(parts[1])
	if err != nil || page < 0 {
		return "", 0, false
	}
	return normalizeTag(parts[0]), page, true
}

func formatClanMemberRole(role Role) string {
	switch strings.ToLower(string(role)) {
	case "leader":
		return "Leader"
	case "coleader":
		return "Co-leader"
	case "admin", "elder":
		return "Elder"
	case "member":
		return "Member"
	default:
		return "Member"
	}
}

func clanMemSortSelectID(tag string, page int) string {
	return fmt.Sprintf("%s%s:%d", clanMemSortPrefix, strings.TrimPrefix(normalizeTag(tag), "#"), page)
}

func parseClanMemSortSelectID(customID string) (tag string, page int, ok bool) {
	if !strings.HasPrefix(customID, clanMemSortPrefix) {
		return "", 0, false
	}
	parts := strings.Split(strings.TrimPrefix(customID, clanMemSortPrefix), ":")
	if len(parts) != 2 {
		return "", 0, false
	}
	page, err := strconv.Atoi(parts[1])
	if err != nil || page < 0 {
		return "", 0, false
	}
	return normalizeTag(parts[0]), page, true
}

func clanPanelComponents(clanTag, activeTab string, state clanPanelState) []discordgo.MessageComponent {
	rows := clanTabComponents(clanTag, activeTab)
	minValues := 1

	switch activeTab {
	case clanTabMembers:
		state.memSort = normalizeClanMemberSort(state.memSort)
		sortOptions := make([]discordgo.SelectMenuOption, 0, len(clanMemberSorts))
		for _, option := range clanMemberSorts {
			sortOptions = append(sortOptions, discordgo.SelectMenuOption{
				Label:       option.label,
				Value:       option.key,
				Description: "Sort members by " + strings.ToLower(option.label),
				Default:     state.memSort == option.key,
			})
		}
		rows = append(rows, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:    clanMemSortSelectID(clanTag, state.memPage),
				Placeholder: "Sort by: " + clanMemberSortLabel(state.memSort),
				MinValues:   &minValues,
				MaxValues:   1,
				Options:     sortOptions,
			},
		}})
		rows = append(rows, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Previous",
				Style:    discordgo.SecondaryButton,
				CustomID: clanMemButtonID("p", clanTag, state.memSort, state.memPage),
				Disabled: state.memPage <= 0,
			},
			discordgo.Button{
				Label:    "Next",
				Style:    discordgo.SecondaryButton,
				CustomID: clanMemButtonID("n", clanTag, state.memSort, state.memPage),
				Disabled: state.memPage >= state.memTotalPages-1,
			},
		}})
	case clanTabWars:
		state.warSort = normalizeClanWarSort(state.warSort)
		sortOptions := make([]discordgo.SelectMenuOption, 0, len(clanWarSorts))
		for _, option := range clanWarSorts {
			sortOptions = append(sortOptions, discordgo.SelectMenuOption{
				Label:       option.label,
				Value:       option.key,
				Description: "Sort wars by " + strings.ToLower(option.label),
				Default:     state.warSort == option.key,
			})
		}
		rows = append(rows, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:    clanWarSortSelectID(clanTag, state.warPage),
				Placeholder: "Sort by: " + clanWarSortLabel(state.warSort),
				MinValues:   &minValues,
				MaxValues:   1,
				Options:     sortOptions,
			},
		}})
		rows = append(rows, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Previous",
				Style:    discordgo.SecondaryButton,
				CustomID: clanWarButtonID("p", clanTag, state.warSort, state.warPage),
				Disabled: state.warPage <= 0,
			},
			discordgo.Button{
				Label:    "Next",
				Style:    discordgo.SecondaryButton,
				CustomID: clanWarButtonID("n", clanTag, state.warSort, state.warPage),
				Disabled: state.warPage >= state.warTotalPages-1,
			},
		}})
	}
	return rows
}

func sendClanPanel(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, clanTag, tab string, state clanPanelState, update bool) {
	responseType := discordgo.InteractionResponseChannelMessageWithSource
	if update {
		responseType = discordgo.InteractionResponseUpdateMessage
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: responseType,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: clanPanelComponents(clanTag, tab, state),
		},
	})
}

func buildClanTabEmbed(clan Clan, tab string, state clanPanelState) (*discordgo.MessageEmbed, clanPanelState) {
	switch tab {
	case clanTabMembers:
		embed, pages := buildClanMembersEmbed(clan, state.memPage, state.memSort)
		state.memTotalPages = pages
		return embed, state
	case clanTabWars:
		embed, pages := buildClanWarsEmbed(clan, state.warPage, state.warSort)
		state.warTotalPages = pages
		return embed, state
	case clanTabCapital:
		return buildClanCapitalEmbed(clan), state
	default:
		return buildClanOverviewEmbed(clan), state
	}
}

func buildClanMembersEmbed(clan Clan, page int, sortKey string) (*discordgo.MessageEmbed, int) {
	result := getClanMembers(clan.Tag)
	if !result.OK {
		return withStatsEmbed(&discordgo.MessageEmbed{
			Title:       clan.Name,
			Description: embedDescriptionWithTag(clan.Tag, "Could not load members: "+result.Error),
		}, clanBadgeURL(clan)), 1
	}

	sortKey = normalizeClanMemberSort(sortKey)

	members := append([]ClanMember(nil), result.Data...)
	sortClanMembers(members, sortKey)

	totalPages := (len(members) + clanMembersPerPage - 1) / clanMembersPerPage
	if totalPages == 0 {
		totalPages = 1
	}
	if page >= totalPages {
		page = totalPages - 1
	}
	if page < 0 {
		page = 0
	}

	start := page * clanMembersPerPage
	end := start + clanMembersPerPage
	if end > len(members) {
		end = len(members)
	}
	pageMembers := members[start:end]

	meta := fmt.Sprintf(
		"### Members (%d)\n-# Sorted by %s · Page %d/%d",
		len(members),
		clanMemberSortLabel(sortKey),
		page+1,
		totalPages,
	)
	if len(pageMembers) == 0 {
		meta += "\nNo members on this page."
	}

	return withStatsEmbed(&discordgo.MessageEmbed{
		Title:       clan.Name,
		Description: embedDescriptionWithTag(clan.Tag, meta),
		Fields:      clanMemberTableFields(pageMembers, sortKey, start),
	}, clanBadgeURL(clan)), totalPages
}

func formatWarSnapshot(snap warSnapshot) string {
	return fmt.Sprintf(
		"**%s** `%s` — `%d`⭐ · `%.0f`%% · `%d` attacks",
		snap.Name, normalizeTag(snap.Tag), snap.Stars, snap.DestructionPercentage, snap.Attacks,
	)
}

func buildClanWarsEmbed(clan Clan, page int, sortKey string) (*discordgo.MessageEmbed, int) {
	sortKey = normalizeClanWarSort(sortKey)
	sections := make([]string, 0, 4)

	current := getClanCurrentWar(clan.Tag)
	if current.OK {
		w := current.Data
		sections = append(sections,
			"### Current War",
			fmt.Sprintf("**State:** %s · **Size:** `%d`v`%d`", w.State, w.TeamSize, w.TeamSize),
			formatWarSnapshot(w.Clan),
			formatWarSnapshot(w.Opponent),
		)
	}

	warLog := getClanWarLog(clan.Tag)
	if !warLog.OK {
		msg := "War log is private or unavailable."
		if !clan.IsWarLogPublic {
			msg = "This clan's war log is set to private."
		} else if warLog.Error != "" {
			msg = warLog.Error
		}
		if len(sections) == 0 {
			sections = append(sections, msg)
		} else {
			sections = append(sections, "### War Log", msg)
		}
		return withStatsEmbed(&discordgo.MessageEmbed{
			Title:       clan.Name,
			Description: embedDescriptionWithTag(clan.Tag, strings.Join(sections, "\n")),
		}, clanBadgeURL(clan)), 1
	}

	entries := append([]warLogEntry(nil), warLog.Data...)
	sortClanWarLog(entries, sortKey)

	totalPages := (len(entries) + clanWarsPerPage - 1) / clanWarsPerPage
	if totalPages == 0 {
		totalPages = 1
	}
	if page >= totalPages {
		page = totalPages - 1
	}
	if page < 0 {
		page = 0
	}

	start := page * clanWarsPerPage
	end := start + clanWarsPerPage
	if end > len(entries) {
		end = len(entries)
	}
	pageEntries := entries[start:end]

	meta := fmt.Sprintf(
		"### War Log (%d)\n-# Sorted by %s · Page %d/%d",
		len(entries),
		clanWarSortLabel(sortKey),
		page+1,
		totalPages,
	)
	if len(pageEntries) == 0 {
		meta += "\nNo war log entries."
	}
	wins, losses, ties := warLogOutcomeCounts(entries)
	if wins+losses+ties > 0 {
		sections = append(sections, fmt.Sprintf("-# Record — `%d` W · `%d` L · `%d` T", wins, losses, ties))
	}
	sections = append(sections, meta)

	embed := withStatsEmbed(&discordgo.MessageEmbed{
		Title:       clan.Name,
		Description: embedDescriptionWithTag(clan.Tag, strings.Join(sections, "\n")),
		Fields:      clanWarTableFields(pageEntries, start),
	}, clanBadgeURL(clan))
	return embed, totalPages
}

func buildClanCapitalEmbed(clan Clan) *discordgo.MessageEmbed {
	capitalLeague := clan.CapitalLeague.Name
	if capitalLeague == "" {
		capitalLeague = "—"
	}
	warLeague := clan.WarLeague.Name
	if warLeague == "" {
		warLeague = "—"
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: "Capital Points", Value: fmt.Sprintf("%d", clan.ClanCapitalPoints), Inline: true},
		{Name: "Builder Base Points", Value: fmt.Sprintf("%d", clan.ClanBuilderBasePoints), Inline: true},
		{Name: "Capital League", Value: capitalLeague, Inline: true},
		{Name: "War League", Value: warLeague, Inline: true},
		{Name: "Capital Hall Level", Value: fmt.Sprintf("%d", clan.ClanCapital.DistrictHallLevel), Inline: true},
	}
	if clan.ClanCapital.Name != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Capital District",
			Value: clan.ClanCapital.Name,
		})
	}

	return withStatsEmbed(&discordgo.MessageEmbed{
		Title:       clan.Name,
		Description: embedDescriptionWithTag(clan.Tag, "### Clan Capital"),
		Fields:      fields,
	}, clanBadgeURL(clan))
}

func handleClanTabButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	tab, tag, ok := parseClanTabButtonID(i.MessageComponentData().CustomID)
	if !ok {
		return
	}

	result := getClanByTag(tag)
	if !result.OK {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{withStatsEmbed(&discordgo.MessageEmbed{
					Title:       "Clan Unavailable",
					Description: result.Error,
				}, "")},
				Components: clanPanelComponents(tag, tab, defaultClanPanelState()),
			},
		})
		return
	}

	upsertKnownClan(result.Data)
	state := defaultClanPanelState()
	if tab == clanTabMembers {
		state.memPage, state.memSort = 0, clanMemberDefaultSort
	}
	if tab == clanTabWars {
		state.warPage, state.warSort = 0, clanWarDefaultSort
	}
	embed, state := buildClanTabEmbed(result.Data, tab, state)
	sendClanPanel(s, i, embed, result.Data.Tag, tab, state, true)
}

func handleClanMembersButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	action, tag, sort, page, ok := parseClanMemButtonID(i.MessageComponentData().CustomID)
	if !ok {
		return
	}

	switch action {
	case "p":
		page--
	case "n":
		page++
	}
	if page < 0 {
		page = 0
	}

	respondClanMembersPanel(s, i, tag, page, sort)
}

func handleClanMembersSortSelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	tag, _, ok := parseClanMemSortSelectID(i.MessageComponentData().CustomID)
	if !ok {
		return
	}

	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return
	}

	respondClanMembersPanel(s, i, tag, 0, normalizeClanMemberSort(values[0]))
}

func respondClanMembersPanel(s *discordgo.Session, i *discordgo.InteractionCreate, tag string, page int, sort string) {
	if page < 0 {
		page = 0
	}

	clanResult := getClanByTag(tag)
	if !clanResult.OK {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{withStatsEmbed(&discordgo.MessageEmbed{
					Title:       "Clan Unavailable",
					Description: clanResult.Error,
				}, "")},
				Components: clanPanelComponents(tag, clanTabMembers, clanPanelState{memSort: sort, memTotalPages: 1, warSort: clanWarDefaultSort, warTotalPages: 1}),
			},
		})
		return
	}

	upsertKnownClan(clanResult.Data)
	state := clanPanelState{memPage: page, memSort: sort, warSort: clanWarDefaultSort, memTotalPages: 1, warTotalPages: 1}
	embed, state := buildClanTabEmbed(clanResult.Data, clanTabMembers, state)
	if state.memPage >= state.memTotalPages {
		state.memPage = state.memTotalPages - 1
		embed, state = buildClanTabEmbed(clanResult.Data, clanTabMembers, state)
	}
	sendClanPanel(s, i, embed, clanResult.Data.Tag, clanTabMembers, state, true)
}

func handleClanWarButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	action, tag, sort, page, ok := parseClanWarButtonID(i.MessageComponentData().CustomID)
	if !ok {
		return
	}

	switch action {
	case "p":
		page--
	case "n":
		page++
	}
	if page < 0 {
		page = 0
	}

	respondClanWarPanel(s, i, tag, page, sort)
}

func handleClanWarSortSelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	tag, _, ok := parseClanWarSortSelectID(i.MessageComponentData().CustomID)
	if !ok {
		return
	}

	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return
	}

	respondClanWarPanel(s, i, tag, 0, normalizeClanWarSort(values[0]))
}

func respondClanWarPanel(s *discordgo.Session, i *discordgo.InteractionCreate, tag string, page int, sort string) {
	if page < 0 {
		page = 0
	}

	clanResult := getClanByTag(tag)
	if !clanResult.OK {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{withStatsEmbed(&discordgo.MessageEmbed{
					Title:       "Clan Unavailable",
					Description: clanResult.Error,
				}, "")},
				Components: clanPanelComponents(tag, clanTabWars, clanPanelState{warSort: sort, memSort: clanMemberDefaultSort, memTotalPages: 1, warTotalPages: 1}),
			},
		})
		return
	}

	upsertKnownClan(clanResult.Data)
	state := clanPanelState{warPage: page, warSort: sort, memSort: clanMemberDefaultSort, memTotalPages: 1, warTotalPages: 1}
	embed, state := buildClanTabEmbed(clanResult.Data, clanTabWars, state)
	if state.warPage >= state.warTotalPages {
		state.warPage = state.warTotalPages - 1
		embed, state = buildClanTabEmbed(clanResult.Data, clanTabWars, state)
	}
	sendClanPanel(s, i, embed, clanResult.Data.Tag, clanTabWars, state, true)
}

func handleClanCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := getCommandContext(i)
	userID := interactionUserID(i)
	clanTag, ok := resolveClanTag(userID, stringOption(ctx.options, "clan"))
	if !ok || clanTag == "" {
		if len(listUserAccounts(userID)) == 0 {
			ephemeralText(s, i, "Verify a player account first with `/verify verify`, or provide a clan tag.")
			return
		}
		ephemeralText(s, i, "None of your linked accounts are in a clan. Provide a clan tag or join a clan in-game.")
		return
	}

	result := getClanByTag(clanTag)
	if !result.OK {
		ephemeralText(s, i, "Failed to fetch clan data: "+result.Error)
		return
	}

	upsertKnownClan(result.Data)
	recordCommandUsage(userID, "clan", result.Data.Tag)
	embed, state := buildClanTabEmbed(result.Data, clanTabOverview, defaultClanPanelState())
	sendClanPanel(s, i, embed, result.Data.Tag, clanTabOverview, state, false)
}

func buildPlayerOverviewEmbed(player Player) *discordgo.MessageEmbed {
	clanName := "No clan"
	if player.Player.Name != "" {
		clanName = player.Player.Name
	}
	return withStatsEmbed(&discordgo.MessageEmbed{
		Title:       player.Name,
		Description: tagSubheading(player.Tag),
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
	}, playerThumbnailURL(player))
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
	return withStatsEmbed(&discordgo.MessageEmbed{
		Title:       player.Name,
		Description: embedDescriptionWithTag(player.Tag, "### Player Equipment\n"+strings.Join(lines, "\n")),
	}, playerThumbnailURL(player))
}

func buildPlayerUpgradeProgressEmbed(player Player) *discordgo.MessageEmbed {
	return withStatsEmbed(&discordgo.MessageEmbed{
		Title:       player.Name,
		Description: embedDescriptionWithTag(player.Tag, "### Upgrade Progress"),
		Fields: []*discordgo.MessageEmbedField{
			makeProgressField("Troops", player.Troops),
			makeProgressField("Heroes", player.Heroes),
			makeProgressField("Spells", player.Spells),
		},
	}, playerThumbnailURL(player))
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
	return withStatsEmbed(&discordgo.MessageEmbed{
		Title:       player.Name,
		Description: embedDescriptionWithTag(player.Tag, "### "+title+"\n"+strings.Join(lines, "\n")),
	}, playerThumbnailURL(player))
}

var playerAchievementSorts = []struct {
	key   string
	label string
}{
	{key: "default", label: "Default Order"},
	{key: "progress-asc", label: "Progress (Low to High)"},
	{key: "progress-desc", label: "Progress (High to Low)"},
}

func normalizePlayerAchSort(sort string) string {
	for _, option := range playerAchievementSorts {
		if option.key == sort {
			return sort
		}
	}
	return playerAchDefaultSort
}

func playerAchSortLabel(sort string) string {
	for _, option := range playerAchievementSorts {
		if option.key == sort {
			return option.label
		}
	}
	return "Default Order"
}

func achievementProgressRatio(achievement PlayerAchievementProgress) float64 {
	if achievement.Target > 0 {
		ratio := float64(achievement.Value) / float64(achievement.Target)
		if ratio > 1 {
			return 1
		}
		return ratio
	}
	return float64(achievement.Stars) / 3
}

func achievementStarTotals(achievements []PlayerAchievementProgress) (earned, possible int) {
	for _, achievement := range achievements {
		earned += achievement.Stars
		possible += 3
	}
	return earned, possible
}

func sortPlayerAchievements(achievements []PlayerAchievementProgress, sortKey string) {
	type indexedAchievement struct {
		achievement PlayerAchievementProgress
		index       int
	}

	items := make([]indexedAchievement, len(achievements))
	for i, achievement := range achievements {
		items[i] = indexedAchievement{achievement: achievement, index: i}
	}

	sortKey = normalizePlayerAchSort(sortKey)
	switch sortKey {
	case "progress-asc":
		sort.Slice(items, func(i, j int) bool {
			ri, rj := achievementProgressRatio(items[i].achievement), achievementProgressRatio(items[j].achievement)
			if ri != rj {
				return ri < rj
			}
			return items[i].index < items[j].index
		})
	case "progress-desc":
		sort.Slice(items, func(i, j int) bool {
			ri, rj := achievementProgressRatio(items[i].achievement), achievementProgressRatio(items[j].achievement)
			if ri != rj {
				return ri > rj
			}
			return items[i].index < items[j].index
		})
	default:
		sort.Slice(items, func(i, j int) bool {
			return items[i].index < items[j].index
		})
	}

	for i, item := range items {
		achievements[i] = item.achievement
	}
}

func formatAchievementLine(index int, achievement PlayerAchievementProgress) string {
	return fmt.Sprintf(
		"**%d.** %s — `%s`/`%s` · `%d` stars",
		index,
		achievement.Name,
		formatCompactNumber(achievement.Value),
		formatCompactNumber(achievement.Target),
		achievement.Stars,
	)
}

func buildPlayerAchievementsEmbed(player Player, page int, sortKey string) (*discordgo.MessageEmbed, int) {
	sortKey = normalizePlayerAchSort(sortKey)
	achievements := append([]PlayerAchievementProgress(nil), player.Achievements...)
	sortPlayerAchievements(achievements, sortKey)

	earnedStars, possibleStars := achievementStarTotals(achievements)

	totalPages := (len(achievements) + playerAchievementsPerPage - 1) / playerAchievementsPerPage
	if totalPages == 0 {
		totalPages = 1
	}
	if page >= totalPages {
		page = totalPages - 1
	}
	if page < 0 {
		page = 0
	}

	start := page * playerAchievementsPerPage
	end := start + playerAchievementsPerPage
	if end > len(achievements) {
		end = len(achievements)
	}
	pageAchievements := achievements[start:end]

	header := fmt.Sprintf(
		"### Achievements\n-# Total: `%s` / `%s` stars · Sorted by %s · Page %d/%d",
		formatCompactNumber(earnedStars),
		formatCompactNumber(possibleStars),
		playerAchSortLabel(sortKey),
		page+1,
		totalPages,
	)

	lines := make([]string, 0, len(pageAchievements)+1)
	lines = append(lines, header)
	if len(pageAchievements) == 0 {
		lines = append(lines, "No achievements found.")
	} else {
		for i, achievement := range pageAchievements {
			lines = append(lines, formatAchievementLine(start+i+1, achievement))
		}
	}

	return withStatsEmbed(&discordgo.MessageEmbed{
		Title:       player.Name,
		Description: embedDescriptionWithTag(player.Tag, strings.Join(lines, "\n")),
	}, playerThumbnailURL(player)), totalPages
}

func playerAchButtonID(action, tag, sort string, page int) string {
	return fmt.Sprintf("%s%s:%s:%d:%s", playerAchPrefix, action, strings.TrimPrefix(normalizeTag(tag), "#"), page, sort)
}

func parsePlayerAchButtonID(customID string) (action, tag, sort string, page int, ok bool) {
	if !strings.HasPrefix(customID, playerAchPrefix) {
		return "", "", "", 0, false
	}
	parts := strings.Split(strings.TrimPrefix(customID, playerAchPrefix), ":")
	if len(parts) != 4 {
		return "", "", "", 0, false
	}
	action = parts[0]
	if action != "p" && action != "n" {
		return "", "", "", 0, false
	}
	page, err := strconv.Atoi(parts[2])
	if err != nil || page < 0 {
		return "", "", "", 0, false
	}
	return action, normalizeTag(parts[1]), normalizePlayerAchSort(parts[3]), page, true
}

func playerAchSortSelectID(tag string, page int) string {
	return fmt.Sprintf("%s%s:%d", playerAchSortPrefix, strings.TrimPrefix(normalizeTag(tag), "#"), page)
}

func parsePlayerAchSortSelectID(customID string) (tag string, page int, ok bool) {
	if !strings.HasPrefix(customID, playerAchSortPrefix) {
		return "", 0, false
	}
	parts := strings.Split(strings.TrimPrefix(customID, playerAchSortPrefix), ":")
	if len(parts) != 2 {
		return "", 0, false
	}
	page, err := strconv.Atoi(parts[1])
	if err != nil || page < 0 {
		return "", 0, false
	}
	return normalizeTag(parts[0]), page, true
}

func playerAchPanelComponents(tag string, page, totalPages int, sort string) []discordgo.MessageComponent {
	sort = normalizePlayerAchSort(sort)
	sortOptions := make([]discordgo.SelectMenuOption, 0, len(playerAchievementSorts))
	for _, option := range playerAchievementSorts {
		sortOptions = append(sortOptions, discordgo.SelectMenuOption{
			Label:       option.label,
			Value:       option.key,
			Description: "Sort achievements by " + strings.ToLower(option.label),
			Default:     sort == option.key,
		})
	}
	minValues := 1
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:    playerAchSortSelectID(tag, page),
				Placeholder: "Sort by: " + playerAchSortLabel(sort),
				MinValues:   &minValues,
				MaxValues:   1,
				Options:     sortOptions,
			},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Previous",
				Style:    discordgo.SecondaryButton,
				CustomID: playerAchButtonID("p", tag, sort, page),
				Disabled: page <= 0,
			},
			discordgo.Button{
				Label:    "Next",
				Style:    discordgo.SecondaryButton,
				CustomID: playerAchButtonID("n", tag, sort, page),
				Disabled: page >= totalPages-1,
			},
		}},
	}
}

func sendPlayerAchievementsPanel(s *discordgo.Session, i *discordgo.InteractionCreate, player Player, page int, sort string, update bool) {
	sort = normalizePlayerAchSort(sort)
	embed, totalPages := buildPlayerAchievementsEmbed(player, page, sort)
	if page >= totalPages {
		page = totalPages - 1
		embed, totalPages = buildPlayerAchievementsEmbed(player, page, sort)
	}

	responseType := discordgo.InteractionResponseChannelMessageWithSource
	if update {
		responseType = discordgo.InteractionResponseUpdateMessage
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: responseType,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: playerAchPanelComponents(player.Tag, page, totalPages, sort),
		},
	})
}

func handlePlayerAchievementsButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	action, tag, sort, page, ok := parsePlayerAchButtonID(i.MessageComponentData().CustomID)
	if !ok {
		return
	}

	switch action {
	case "p":
		page--
	case "n":
		page++
	}
	if page < 0 {
		page = 0
	}

	respondPlayerAchievementsPanel(s, i, tag, page, sort)
}

func handlePlayerAchievementsSortSelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	tag, _, ok := parsePlayerAchSortSelectID(i.MessageComponentData().CustomID)
	if !ok {
		return
	}

	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return
	}

	respondPlayerAchievementsPanel(s, i, tag, 0, normalizePlayerAchSort(values[0]))
}

func respondPlayerAchievementsPanel(s *discordgo.Session, i *discordgo.InteractionCreate, tag string, page int, sort string) {
	if page < 0 {
		page = 0
	}

	result := getPlayerByTag(tag)
	if !result.OK {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{withStatsEmbed(&discordgo.MessageEmbed{
					Title:       "Player Unavailable",
					Description: result.Error,
				}, "")},
			},
		})
		return
	}

	sendPlayerAchievementsPanel(s, i, result.Data, page, sort, true)
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
	return withStatsEmbed(&discordgo.MessageEmbed{
		Title:       "Linked Accounts",
		Description: strings.Join(lines, "\n"),
	}, "")
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
		ephemeralText(s, i, "Provide a player tag/name or set a main account with `/verify set-main`.")
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
	"clan":                    "/clan [clan]",
	"help":                    "/help [command]",
	"player profile":          "/player profile [player]",
	"player achievements":     "/player achievements [player]",
	"player equipment":        "/player equipment [player]",
	"player heroes":           "/player heroes [player]",
	"player spells":           "/player spells [player]",
	"player troops":           "/player troops [player]",
	"player upgrade-progress": "/player upgrade-progress [player]",
	"verify verify":           "/verify verify player",
	"verify list":             "/verify list",
	"verify remove":           "/verify remove player",
	"verify set-main":         "/verify set-main player",
}

func getHelpCommandNames() []string {
	names := make([]string, 0, len(helpUsageByCommand))
	for name := range helpUsageByCommand {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func formatHelpCommandList(commands []string) string {
	lines := make([]string, 0, len(commands))
	for _, usage := range commands {
		lines = append(lines, "- `"+usage+"`")
	}
	return strings.Join(lines, "\n")
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
	if subcommand == "verify" {
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
			Description: "Player stats and progression.",
			Options: []*discordgo.ApplicationCommandOption{
				playerSubcommand("profile", "Shows a player summary."),
				playerSubcommand("equipment", "Shows hero equipment levels."),
				playerSubcommand("heroes", "Shows hero levels."),
				playerSubcommand("troops", "Shows troop levels."),
				playerSubcommand("spells", "Shows spell levels."),
				playerSubcommand("achievements", "Shows achievement progress."),
				playerSubcommand("upgrade-progress", "Shows troops, heroes, and spells progression."),
			},
		},
		{
			Name:        "verify",
			Description: "Manage linked player accounts.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
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

			if commandName != "" {
				usage, found := helpUsageByCommand[commandName]
				if !found {
					ephemeralText(s, i, "Unknown command. Use autocomplete for valid commands.")
					return
				}
				respondWithEmbed(s, i, withStatsEmbed(&discordgo.MessageEmbed{
					Title:       "Help: " + commandName,
					Description: formatHelpCommandList([]string{usage}),
				}, ""))
				return
			}

			commands := make([]string, 0, len(helpUsageByCommand))
			for _, name := range getHelpCommandNames() {
				commands = append(commands, helpUsageByCommand[name])
			}
			respondWithEmbed(s, i, withStatsEmbed(&discordgo.MessageEmbed{
				Title:       "Available Commands",
				Description: formatHelpCommandList(commands),
			}, ""))
		},
		"clan": handleClanCommand,
		"player": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			ctx := getCommandContext(i)
			player, ok := resolveAndFetchPlayer(s, i)
			if !ok {
				return
			}

			switch ctx.subcommand {
			case "profile":
				respondWithEmbed(s, i, buildPlayerOverviewEmbed(player))
			case "equipment":
				respondWithEmbed(s, i, buildPlayerEquipmentProgressEmbed(player))
			case "heroes":
				respondWithEmbed(s, i, buildPlayerItemsEmbed(player, "Heroes", player.Heroes, "No heroes found."))
			case "troops":
				respondWithEmbed(s, i, buildPlayerItemsEmbed(player, "Troops", player.Troops, "No troops found."))
			case "spells":
				respondWithEmbed(s, i, buildPlayerItemsEmbed(player, "Spells", player.Spells, "No spells found."))
			case "achievements":
				sendPlayerAchievementsPanel(s, i, player, 0, playerAchDefaultSort, false)
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
			case "verify":
				playerTag, ok := resolvePlayerTag(userID, stringOption(ctx.options, "player"))
				if !ok || playerTag == "" {
					ephemeralText(s, i, "Provide a valid player tag.")
					return
				}
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseModal,
					Data: &discordgo.InteractionResponseData{
						CustomID: verifyTokenModalPrefix + playerTag,
						Title:    botWatermark + " · Verify Account",
						Components: []discordgo.MessageComponent{
							discordgo.ActionsRow{
								Components: []discordgo.MessageComponent{
									discordgo.TextInput{
										CustomID:    verifyTokenInputID,
										Label:       "In-Game API Token",
										Style:       discordgo.TextInputShort,
										Placeholder: "Settings → More Settings → API Token",
										Required:    true,
										MinLength:   1,
										MaxLength:   64,
									},
								},
							},
						},
					},
				})
			case "list":
				accounts := listUserAccounts(userID)
				mainTag, _ := getUserMainAccount(userID)
				if len(accounts) == 0 {
					respondWithEphemeralEmbed(s, i, buildStatusEmbed(
						"Linked Accounts",
						"No linked accounts yet. Use `/verify verify` with a player tag to start.",
						"",
					))
					return
				}
				respondWithEphemeralEmbed(s, i, buildVerifyListEmbed(accounts, mainTag))
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
				respondWithEphemeralEmbed(s, i, buildStatusEmbed(
					"Account Removed",
					"Removed linked account `"+playerTag+"`.",
					"",
				))
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
				respondWithEphemeralEmbed(s, i, buildStatusEmbed(
					"Main Account Updated",
					"Set `"+playerTag+"` as your main account.",
					"",
				))
			default:
				ephemeralText(s, i, "Unsupported verify subcommand.")
			}
		},
	}
)
