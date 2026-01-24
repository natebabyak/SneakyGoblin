package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Player struct {
	Tag                      int    `json:"tag"`
	Name                     string `json:"name"`
	TownHallLevel            int    `json:"townHallLevel"`
	ExpLevel                 int    `json:"expLevel"`
	Trophies                 int    `json:"trophies"`
	BestTrophies             int    `json:"bestTrophies"`
	WarStars                 int    `json:"warStars"`
	AttackWins               int    `json:"attackWins"`
	DefenseWins              int    `json:"defenseWins"`
	BuilderHallLevel         int    `json:"builderHallLevel"`
	BuilderBaseTrophies      int    `json:"builderBaseTrophies"`
	BestBuilderBaseTrophies  int    `json:"bestBuilderBaseTrophies"`
	Role                     string `json:"role"`
	WarPreference            string `json:"warPreference"`
	Donations                int    `json:"donations"`
	DonationsReceived        int    `json:"donationsReceived"`
	ClanCapitalContributions int    `json:"clanCapitalContributions"`
	Clan                     struct {
		Tag       string `json:"tag"`
		Name      string `json:"name"`
		ClanLevel int    `json:"clanLevel"`
		BadgeUrls struct {
			Small  string `json:"small"`
			Large  string `json:"large"`
			Medium string `json:"medium"`
		} `json:"badgeUrls"`
	} `json:"clan"`
	LeagueTier struct {
		Id       int    `json:"id"`
		Name     string `json:"name"`
		IconUrls struct {
			Small string `json:"small"`
			Large string `json:"large"`
		} `json:"iconUrls"`
	} `json:"leagueTier"`
	BuilderBaseLeague struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	} `json:"builderBaseLeague"`
	Achievements []struct {
		Name           string `json:"name"`
		Stars          int    `json:"stars"`
		Value          int    `json:"value"`
		Target         int    `json:"target"`
		Info           string `json:"info"`
		CompletionInfo string `json:"completionInfo"`
		Village        string `json:"village"`
	} `json:"achievements"`
	PlayerHouse struct {
		Elements []struct {
			Type string `json:"type"`
			Id   int    `json:"id"`
		} `json:"elements"`
	} `json:"playerHouse"`
	Labels []struct {
		Id       int    `json:"id"`
		Name     string `json:"name"`
		IconUrls struct {
			Small  string `json:"small"`
			Medium string `json:"medium"`
		} `json:"iconUrls"`
	} `json:"labels"`
	Troops []struct {
		Name     string `json:"name"`
		Level    int    `json:"level"`
		MaxLevel int    `json:"maxLevel"`
		Village  string `json:"village"`
	} `json:"troops"`
	Heroes []struct {
		Name      string `json:"name"`
		Level     int    `json:"level"`
		MaxLevel  int    `json:"maxLevel"`
		Equipment []struct {
			Name     string `json:"name"`
			Level    int    `json:"level"`
			MaxLevel int    `json:"maxLevel"`
			Village  string `json:"village"`
		} `json:"equipment"`
		Village string `json:"village"`
	} `json:"heroes"`
	HeroEquipment []struct {
		Name     string `json:"name"`
		Level    int    `json:"level"`
		MaxLevel int    `json:"maxLevel"`
		Village  string `json:"village"`
	} `json:"heroEquipment"`
	Spells []struct {
		Name     string `json:"name"`
		Level    int    `json:"level"`
		MaxLevel int    `json:"maxLevel"`
		Village  string `json:"village"`
	} `json:"spells"`
}

var (
	token string
)

func init() {
	godotenv.Load()
	token = os.Getenv("API_TOKEN")
}

func get_player_by_tag(tag string) (*Player, error) {
	encodedTag := url.PathEscape(tag)

	req, err := http.NewRequest(
		"GET",
		"https://api.clashofclans.com/v1/players/"+encodedTag,
		nil,
	)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", res.Status)
	}

	var player Player
	if err := json.NewDecoder(res.Body).Decode(&player); err != nil {
		return nil, err
	}

	return &player, nil
}
