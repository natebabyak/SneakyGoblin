package models

type LeagueTier struct {
	Name     string
	Id       int
	IconUrls struct {
		Small string
		Large string
	}
}
