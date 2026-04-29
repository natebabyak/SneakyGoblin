package models

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
