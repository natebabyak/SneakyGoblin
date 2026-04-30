package api

import (
	"encoding/json"
	"log"

	"main/models"
)

func GetPlayerByTag(tag string) models.Player {
	data := get(baseURL + "players/" + formatTag(tag))
	var player models.Player
	if err := json.Unmarshal(data, &player); err != nil {
		log.Println("failed to parse player response:", err)
		return models.Player{}
	}
	return player
}
