package models

type Clan struct {
	memberList []struct {
		league struct {
			name     string
			id       int
			iconUrls struct {
				small string
				tiny  string
			}
		}
		leagueTier struct {
			name     string
			id       int
			iconUrls struct {
				small string
				large string
			}
		}
		builderBaseLeague struct {
			name string
			id   int
		}
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
		playerHouse         struct {
			elements []struct {
				id   int
				Type string
			}
		}
	}
	warLeague struct {
		id   int
		name string
	}
	capitalLeague struct {
		id   int
		name string
	}
	tag          string
	clanLevel    int
	warWinStreak int
	warWins      int
	warTies      int
	warLosses    int
	clanPoints   int
	chatLanguage struct {
		name         string
		id           int
		languageCode string
	}
	warFrequency                string
	clanBuilderBasePoints       int
	clanCapitalPoints           int
	requiredTrophies            int
	requiredBuilderBaseTrophies int
	requiredTownhallLevel       int
	isFamilyFriendly            bool
	isWarLogPublic              bool
	labels                      []struct {
		name     string
		id       int
		iconUrls struct {
			small  string
			medium string
		}
	}
	name     string
	location struct {
		localizedName string
		id            int
		name          string
		isCountry     bool
		countryCode   string
	}
	Type        string
	members     int
	description string
	clanCapital struct {
		capitalHallLevel int
		districts        []struct {
			name              string
			id                int
			districtHallLevel int
		}
	}
	badgeUrls struct {
		small  string
		large  string
		medium string
	}
}
