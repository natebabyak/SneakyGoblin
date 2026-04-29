package models

type PlayerLegendStatistics struct {
	CurrentSeason             []LegendLeagueTournamentSeasonResult
	BestSeason                []LegendLeagueTournamentSeasonResult
	PreviousSeason            []LegendLeagueTournamentSeasonResult
	PreviousBuilderBaseSeason []LegendLeagueTournamentSeasonResult
	BestBuilderBaseSeason     []LegendLeagueTournamentSeasonResult
	LegendTrophies            int
}
