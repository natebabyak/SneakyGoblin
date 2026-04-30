package main

type ClanType string
type PlayerHouseElementType string
type Role string
type Village string
type WarFrequency string
type WarPreference string

const (
	OPEN        ClanType = "OPEN"
	INVITE_ONLY ClanType = "INVITE_ONLY"
	CLOSED      ClanType = "CLOSED"
)

const (
	GROUND PlayerHouseElementType = "GROUND"
	ROOF   PlayerHouseElementType = "ROOF"
	FOOT   PlayerHouseElementType = "FOOT"
	DECO   PlayerHouseElementType = "DECO"
)

const (
	NOT_MEMBER Role = "NOT_MEMBER"
	MEMBER     Role = "MEMBER"
	LEADER     Role = "LEADER"
	ADMIN      Role = "ADMIN"
	COLEADER   Role = "COLEADER"
)

const (
	HOME_VILLAGE Village = "HOME_VILLAGE"
	BUILDER_BASE Village = "BUILDER_BASE"
	CLAN_CAPITAL Village = "CLAN_CAPITAL"
)

const (
	UNKNOWN                 WarFrequency = "UNKNOWN"
	ALWAYS                  WarFrequency = "ALWAYS"
	MORE_THAN_ONCE_PER_WEEK WarFrequency = "MORE_THAN_ONCE_PER_WEEK"
	ONCE_PER_WEEK           WarFrequency = "ONCE_PER_WEEK"
	LESS_THAN_ONCE_PER_WEEK WarFrequency = "LESS_THAN_ONCE_PER_WEEK"
	NEVER                   WarFrequency = "NEVER"
	ANY                     WarFrequency = "ANY"
)

const (
	OUT WarPreference = "OUT"
	IN  WarPreference = "IN"
)

type BuilderBaseLeague struct {
	Name string
	Id   int
}

type CapitalLeague struct {
	Name string
	Id   int
}

type ClanCapital struct {
	CapitalHallLevel int
	Districts        []ClanDistrictData
}

type ClanDistrictData struct {
	Name              string
	Id                int
	DistrictHallLevel int
}

type ClanMember struct {
	League              League
	LeagueTier          LeagueTier
	BuilderBaseLeague   BuilderBaseLeague
	Tag                 string
	Name                string
	Role                Role
	TownHallLevel       int
	ExpLevel            int
	ClanRank            int
	PreviousClanRank    int
	Donations           int
	DonationsReceived   int
	Trophies            int
	BuilderBaseTrophies int
	PlayerHouse         PlayerHouse
}

type Clan struct {
	MemberList                  []ClanMember
	WarLeague                   WarLeague
	CapitalLeague               CapitalLeague
	Tag                         string
	ClanLevel                   int
	WarWinStreak                int
	WarWins                     int
	WarTies                     int
	WarLosses                   int
	ClanPoints                  int
	ChatLanguage                Language
	WarFrequency                WarFrequency
	ClanBuilderBasePoints       int
	ClanCapitalPoints           int
	RequiredTrophies            int
	RequiredBuilderBaseTrophies int
	RequiredTownhallLevel       int
	IsFamilyFriendly            bool
	IsWarLogPublic              bool
	Labels                      []Label
	Name                        string
	Location                    Location
	Type                        ClanType
	Members                     int
	Description                 string
	ClanCapital                 ClanDistrictData
	BadgeUrls                   struct {
		Small  string
		Large  string
		Medium string
	}
}

type Label struct {
	Name     string
	Id       int
	IconUrls struct {
		Small  string
		Medium string
	}
}

type Language struct {
	Name         string
	Id           int
	LanguageCode string
}

type LeagueTier struct {
	Name     string
	Id       int
	IconUrls struct {
		Small string
		Large string
	}
}

type League struct {
	Name     string
	Id       int
	IconUrls struct {
		Small string
		Tiny  string
	}
}

type LegendLeagueTournamentSeasonResult struct {
	Trophies int
	Id       string
	Rank     int
}

type Location struct {
	LocalizedName string
	Id            int
	Name          string
	IsCountry     bool
	CountryCode   string
}

type PlayerAchievementProgress struct {
	Stars          int
	Value          int
	Name           string
	Target         int
	Info           string
	CompletionInfo string
	Village        Village
}

type PlayerClan struct {
	Tag       string
	ClanLevel int
	Name      string
	BadgeUrls struct {
		Small  string
		Large  string
		Medium string
	}
}

type PlayerHouse struct {
	Elements []struct {
		Type PlayerHouseElementType
		Id   int
	}
}

type PlayerItemLevel struct {
	Level              int
	Name               string
	MaxLevel           int
	Village            Village
	SuperTroopIsActive bool
	Equipment          []struct {
		Name     string
		Level    int
		MaxLevel int
		Village  Village
	}
}

type PlayerLegendStatistics struct {
	CurrentSeason             []LegendLeagueTournamentSeasonResult
	BestSeason                []LegendLeagueTournamentSeasonResult
	PreviousSeason            []LegendLeagueTournamentSeasonResult
	PreviousBuilderBaseSeason []LegendLeagueTournamentSeasonResult
	BestBuilderBaseSeason     []LegendLeagueTournamentSeasonResult
	LegendTrophies            int
}

type WarLeague struct {
	Name string
	Id   int
}

type Player struct {
	Player                   PlayerClan
	League                   League
	LeagueTier               LeagueTier
	BuilderBaseLeague        BuilderBaseLeague
	Role                     Role
	WarPreference            WarPreference
	AttackWins               int
	DefenseWins              int
	TownHallLevel            int
	TownHallWeaponLevel      int
	LegendStatistics         PlayerLegendStatistics
	Troops                   []PlayerItemLevel
	Heroes                   []PlayerItemLevel
	HeroEquipment            []PlayerItemLevel
	Spells                   []PlayerItemLevel
	Labels                   []Label
	Tag                      string
	Name                     string
	ExpLevel                 int
	Trophies                 int
	BestTrophies             int
	Donations                int
	DonationsReceived        int
	BuilderHallLevel         int
	BuilderBaseTrophies      int
	BestBuilderBaseTrophies  int
	WarStars                 int
	Achievements             []PlayerAchievementProgress
	ClanCapitalContributions int
	PlayerHouse              PlayerHouse
	CurrentLeagueGroupTag    string
	CurrentLeagueSeasonId    int
	PreviousLeagueGroupTag   string
	PreviousLeagueSeasonId   int
}
