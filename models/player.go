package models

type Player struct {
	player                   PlayerClan
	league                   League
	leagueTier               LeagueTier
	builderBaseLeague        BuilderBaseLeague
	role                     Role
	warPreference            WarPreference
	attackWins               int
	defenseWins              int
	townHallLevel            int
	townHallWeaponLevel      int
	legendStatistics         PlayerLegendStatistics
	troops                   []PlayerItemLevel
	heroes                   []PlayerItemLevel
	heroEquipment            []PlayerItemLevel
	spells                   []PlayerItemLevel
	labels                   []Label
	tag                      string
	name                     string
	expLevel                 int
	trophies                 int
	bestTrophies             int
	donations                int
	donationsReceived        int
	builderHallLevel         int
	builderBaseTrophies      int
	bestBuilderBaseTrophies  int
	warStars                 int
	achievements             []PlayerAchievementProgress
	clanCapitalContributions int
	playerHouse              PlayerHouse
	currentLeagueGroupTag    string
	currentLeagueSeasonId    int
	previousLeagueGroupTag   string
	previousLeagueSeasonId   int
}
