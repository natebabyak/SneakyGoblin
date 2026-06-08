package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const baseURL = "https://api.clashofclans.com/v1/"

var (
	httpClient = &http.Client{Timeout: 10 * time.Second}
)

type APIResult[T any] struct {
	Data       T
	OK         bool
	StatusCode int
	Error      string
}

func normalizeTag(raw string) string {
	tag := strings.ToUpper(strings.TrimSpace(raw))
	if tag == "" {
		return ""
	}
	if strings.HasPrefix(tag, "#") {
		return tag
	}
	return "#" + tag
}

func formatTag(tag string) string {
	normalized := normalizeTag(tag)
	if normalized == "" {
		return ""
	}
	return url.PathEscape(normalized)
}

func doRequest(method, rawURL string, body []byte) ([]byte, int, string, bool) {
	var requestBody io.Reader
	if len(body) > 0 {
		requestBody = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, rawURL, requestBody)
	if err != nil {
		log.Println("failed to build request:", err)
		return nil, 0, "failed to build request", false
	}

	if cocToken == "" {
		log.Println("COC_TOKEN is empty; set it in .env for Clash API requests")
		return nil, 0, "COC token is not configured", false
	}
	req.Header.Set("Authorization", "Bearer "+cocToken)
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Println("request failed:", err)
		return nil, 0, "failed to reach Clash API", false
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("failed to read response body:", err)
		return nil, resp.StatusCode, "failed to read Clash API response", false
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		log.Printf("non-2xx status %d from %s", resp.StatusCode, rawURL)
		return nil, resp.StatusCode, parseAPIError(data), false
	}

	return data, resp.StatusCode, "", true
}

func get(rawURL string) ([]byte, int, string, bool) {
	return doRequest(http.MethodGet, rawURL, nil)
}

func postJSON(rawURL string, payload any) ([]byte, int, string, bool) {
	body, err := json.Marshal(payload)
	if err != nil {
		log.Println("failed to encode request body:", err)
		return nil, 0, "failed to encode request body", false
	}
	return doRequest(http.MethodPost, rawURL, body)
}

func parseAPIError(data []byte) string {
	var e struct {
		Reason  string `json:"reason"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(data, &e); err == nil {
		if e.Message != "" {
			return e.Message
		}
		if e.Reason != "" {
			return e.Reason
		}
	}
	return "Clash API rejected the request"
}

func getClanByTag(tag string) APIResult[Clan] {
	data, statusCode, reason, ok := get(baseURL + "clans/" + formatTag(tag))
	if !ok {
		return APIResult[Clan]{OK: false, StatusCode: statusCode, Error: reason}
	}

	var clan Clan
	if err := json.Unmarshal(data, &clan); err != nil {
		log.Println("failed to parse clan response:", err)
		return APIResult[Clan]{OK: false, StatusCode: statusCode, Error: "failed to parse clan response"}
	}
	return APIResult[Clan]{Data: clan, OK: true, StatusCode: statusCode}
}

type clanMembersResponse struct {
	Items []ClanMember `json:"items"`
}

type warSnapshot struct {
	Name                  string  `json:"name"`
	Tag                   string  `json:"tag"`
	Stars                 int     `json:"stars"`
	DestructionPercentage float64 `json:"destructionPercentage"`
	Attacks               int     `json:"attacks"`
	ClanLevel             int     `json:"clanLevel"`
}

type warLogEntry struct {
	Result   string      `json:"result"`
	EndTime  string      `json:"endTime"`
	TeamSize int         `json:"teamSize"`
	Clan     warSnapshot `json:"clan"`
	Opponent warSnapshot `json:"opponent"`
}

type warLogResponse struct {
	Items []warLogEntry `json:"items"`
}

type currentWarResponse struct {
	State     string      `json:"state"`
	TeamSize  int         `json:"teamSize"`
	Clan      warSnapshot `json:"clan"`
	Opponent  warSnapshot `json:"opponent"`
	StartTime string      `json:"startTime"`
	EndTime   string      `json:"endTime"`
}

func getClanMembers(tag string) APIResult[[]ClanMember] {
	data, statusCode, reason, ok := get(baseURL + "clans/" + formatTag(tag) + "/members")
	if !ok {
		return APIResult[[]ClanMember]{OK: false, StatusCode: statusCode, Error: reason}
	}

	var members clanMembersResponse
	if err := json.Unmarshal(data, &members); err != nil {
		log.Println("failed to parse clan members response:", err)
		return APIResult[[]ClanMember]{OK: false, StatusCode: statusCode, Error: "failed to parse clan members response"}
	}
	for i := range members.Items {
		patchClanMemberFields(data, &members.Items[i], i)
	}
	return APIResult[[]ClanMember]{Data: members.Items, OK: true, StatusCode: statusCode}
}

func patchClanMemberFields(data []byte, member *ClanMember, index int) {
	var payload struct {
		Items []struct {
			Tag        string     `json:"tag"`
			League     League     `json:"league"`
			LeagueTier LeagueTier `json:"leagueTier"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || index >= len(payload.Items) {
		return
	}
	item := payload.Items[index]
	if strings.TrimSpace(member.Tag) == "" {
		member.Tag = item.Tag
	}
	if member.League.Name == "" && member.League.Id == 0 {
		member.League = item.League
	}
	if member.LeagueTier.Name == "" && member.LeagueTier.Id == 0 {
		member.LeagueTier = item.LeagueTier
	}
}

func getClanWarLog(tag string) APIResult[[]warLogEntry] {
	data, statusCode, reason, ok := get(baseURL + "clans/" + formatTag(tag) + "/warlog")
	if !ok {
		return APIResult[[]warLogEntry]{OK: false, StatusCode: statusCode, Error: reason}
	}

	var warLog warLogResponse
	if err := json.Unmarshal(data, &warLog); err != nil {
		log.Println("failed to parse clan war log response:", err)
		return APIResult[[]warLogEntry]{OK: false, StatusCode: statusCode, Error: "failed to parse clan war log response"}
	}
	return APIResult[[]warLogEntry]{Data: warLog.Items, OK: true, StatusCode: statusCode}
}

func getClanCurrentWar(tag string) APIResult[currentWarResponse] {
	data, statusCode, reason, ok := get(baseURL + "clans/" + formatTag(tag) + "/currentwar")
	if !ok {
		return APIResult[currentWarResponse]{OK: false, StatusCode: statusCode, Error: reason}
	}

	var war currentWarResponse
	if err := json.Unmarshal(data, &war); err != nil {
		log.Println("failed to parse current war response:", err)
		return APIResult[currentWarResponse]{OK: false, StatusCode: statusCode, Error: "failed to parse current war response"}
	}
	return APIResult[currentWarResponse]{Data: war, OK: true, StatusCode: statusCode}
}

func getPlayerByTag(tag string) APIResult[Player] {
	data, statusCode, reason, ok := get(baseURL + "players/" + formatTag(tag))
	if !ok {
		return APIResult[Player]{OK: false, StatusCode: statusCode, Error: reason}
	}

	var player Player
	if err := json.Unmarshal(data, &player); err != nil {
		log.Println("failed to parse player response:", err)
		return APIResult[Player]{OK: false, StatusCode: statusCode, Error: "failed to parse player response"}
	}
	patchPlayerAPIFields(data, &player)
	return APIResult[Player]{Data: player, OK: true, StatusCode: statusCode}
}

// patchPlayerAPIFields fills fields the Player struct cannot map by name alone (e.g. clan).
func patchPlayerAPIFields(data []byte, player *Player) {
	var aux struct {
		Clan PlayerClan `json:"clan"`
		Role Role       `json:"role"`
		Tag  string     `json:"tag"`
		Name string     `json:"name"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return
	}
	if strings.TrimSpace(player.Player.Tag) == "" && strings.TrimSpace(aux.Clan.Tag) != "" {
		player.Player = aux.Clan
	}
	if player.Role == "" && aux.Role != "" {
		player.Role = aux.Role
	}
	if strings.TrimSpace(player.Tag) == "" {
		player.Tag = aux.Tag
	}
	if strings.TrimSpace(player.Name) == "" {
		player.Name = aux.Name
	}
}

type verifyTokenRequest struct {
	Token string `json:"token"`
}

type verifyTokenResponse struct {
	Tag    string `json:"tag"`
	Token  string `json:"token"`
	Status string `json:"status"`
}

func verifyPlayerToken(tag, token string) APIResult[verifyTokenResponse] {
	data, statusCode, reason, ok := postJSON(
		baseURL+"players/"+formatTag(tag)+"/verifytoken",
		verifyTokenRequest{Token: strings.TrimSpace(token)},
	)
	if !ok {
		return APIResult[verifyTokenResponse]{OK: false, StatusCode: statusCode, Error: reason}
	}

	var verification verifyTokenResponse
	if err := json.Unmarshal(data, &verification); err != nil {
		log.Println("failed to parse verify token response:", err)
		return APIResult[verifyTokenResponse]{OK: false, StatusCode: statusCode, Error: "failed to parse verify token response"}
	}
	return APIResult[verifyTokenResponse]{Data: verification, OK: true, StatusCode: statusCode}
}
