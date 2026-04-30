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
	token      string
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

func get(rawURL string) []byte {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return data
}

func getClanByTag(tag string) Clan {
	data := get(baseURL + "clans/" + formatTag(tag))
	var clan Clan
	if err := json.Unmarshal(data, &clan); err != nil {
		log.Println("failed to parse clan response:", err)
		return Clan{}
	}
	return clan
}

func getPlayerByTag(tag string) Player {
	data := get(baseURL + "players/" + formatTag(tag))
	var player Player
	if err := json.Unmarshal(data, &player); err != nil {
		log.Println("failed to parse player response:", err)
		return Player{}
	}
	return player
}
