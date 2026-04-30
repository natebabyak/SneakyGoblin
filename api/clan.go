package api

import (
	"encoding/json"
	"log"

	"main/models"
)

func GetClanByTag(tag string) models.Clan {
	data := get(baseURL + "clans/" + formatTag(tag))
	var clan models.Clan
	if err := json.Unmarshal(data, &clan); err != nil {
		log.Println("failed to parse clan response:", err)
		return models.Clan{}
	}
	return clan
}
