package models

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
