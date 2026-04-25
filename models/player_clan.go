package models

type PlayerClan struct {
	tag       string
	clanLevel int
	name      string
	badgeUrls struct {
		small  string
		large  string
		medium string
	}
}
