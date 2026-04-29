package models

type Location struct {
	LocalizedName string
	Id            int
	Name          string
	IsCountry     bool
	CountryCode   string
}
