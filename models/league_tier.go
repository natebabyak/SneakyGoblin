package models

type LeagueTier struct {
	name     string
	id       int
	iconUrls struct {
		small string
		large string
	}
}
