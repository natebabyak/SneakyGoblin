package main

import (
	"encoding/json"
	"io"
	"log"
	"main/models"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/joho/godotenv"
)

var (
	client http.Client = http.Client{
		Timeout: 10 * time.Second,
	}
	token string
)

func init() {
	godotenv.Load()
	token = os.Getenv("API_TOKEN")

}

func formatTag(tag string) string {
	if tag[0] == '#' {
		return url.PathEscape(tag)
	} else {
		return url.PathEscape("#") + tag
	}
}

func Get(url string) []byte {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Accept", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
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

const baseUrl = "https://api.clashofclans.com/v1/"

func GetClanByTag(tag string) models.Clan {
	data := Get(baseUrl + "clans/" + formatTag(tag))
	var clan models.Clan
	if err := json.Unmarshal(data, &clan); err != nil {
		log.Println("failed to parse clan response:", err)
		return models.Clan{}
	}
	return clan
}

func GetPlayerByTag(tag string) models.Player {
	data := Get(baseUrl + "players/" + formatTag(tag))
	var player models.Player
	if err := json.Unmarshal(data, &player); err != nil {
		log.Println("failed to parse player response:", err)
		return models.Player{}
	}
	return player
}
