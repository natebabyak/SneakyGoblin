package models

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
