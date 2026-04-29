package models

type PlayerClan struct {
	Tag       string
	ClanLevel int
	Name      string
	BadgeUrls struct {
		Small  string
		Large  string
		Medium string
	}
}
