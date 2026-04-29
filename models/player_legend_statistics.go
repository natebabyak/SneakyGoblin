package models

type PlayerLegendStatistics struct {
	currentSeason             []LegendLeagueTournamentSeasonResult
	bestSeason                []LegendLeagueTournamentSeasonResult
	previousSeason            []LegendLeagueTournamentSeasonResult
	previousBuilderBaseSeason []LegendLeagueTournamentSeasonResult
	bestBuilderBaseSeason     []LegendLeagueTournamentSeasonResult
	legendTrophies            int
}
