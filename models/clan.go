package models

type Clan struct {
	memberList                  []ClanMember
	warLeague                   WarLeague
	capitalLeague               CapitalLeague
	tag                         string
	clanLevel                   int
	warWinStreak                int
	warWins                     int
	warTies                     int
	warLosses                   int
	clanPoints                  int
	chatLanguage                Language
	warFrequency                WarFrequency
	clanBuilderBasePoints       int
	clanCapitalPoints           int
	requiredTrophies            int
	requiredBuilderBaseTrophies int
	requiredTownhallLevel       int
	isFamilyFriendly            bool
	isWarLogPublic              bool
	labels                      []Label
	name                        string
	location                    Location
	Type                        ClanType
	members                     int
	description                 string
	clanCapital                 ClanDistrictData
	badgeUrls                   struct {
		small  string
		large  string
		medium string
	}
}
