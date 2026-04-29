package models

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
