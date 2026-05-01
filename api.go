package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

const baseURL = "https://api.clashofclans.com/v1/"

var (
	httpClient = &http.Client{Timeout: 10 * time.Second}
)

func formatTag(tag string) string {
	if tag == "" {
		return ""
	}
	if tag[0] == '#' {
		return url.PathEscape(tag)
	}
	return url.PathEscape("#") + tag
}

func get(rawURL string) ([]byte, bool) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		log.Println("failed to build request:", err)
		return nil, false
	}

	if cocToken == "" {
		log.Println("COC_TOKEN is empty; set it in .env for Clash API requests")
		return nil, false
	}
	req.Header.Set("Authorization", "Bearer "+cocToken)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Println("request failed:", err)
		return nil, false
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("failed to read response body:", err)
		return nil, false
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		log.Printf("non-2xx status %d from %s", resp.StatusCode, rawURL)
		return nil, false
	}

	return data, true
}

func getClanByTag(tag string) (Clan, bool) {
	data, ok := get(baseURL + "clans/" + formatTag(tag))
	if !ok {
		return Clan{}, false
	}

	var clan Clan
	if err := json.Unmarshal(data, &clan); err != nil {
		log.Println("failed to parse clan response:", err)
		return Clan{}, false
	}
	return clan, true
}

func getPlayerByTag(tag string) (Player, bool) {
	data, ok := get(baseURL + "players/" + formatTag(tag))
	if !ok {
		return Player{}, false
	}

	var player Player
	if err := json.Unmarshal(data, &player); err != nil {
		log.Println("failed to parse player response:", err)
		return Player{}, false
	}
	return player, true
}
