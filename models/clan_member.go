package models

type ClanMember struct {
	league              League
	leagueTier          LeagueTier
	builderBaseLeague   BuilderBaseLeague
	tag                 string
	name                string
	role                string
	townHallLevel       int
	expLevel            int
	clanRank            int
	previousClanRank    int
	donations           int
	donationsReceived   int
	trophies            int
	builderBaseTrophies int
	playerHouse         PlayerHouse
}
