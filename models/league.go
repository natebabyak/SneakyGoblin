package models

type League struct {
	Name     string
	Id       int
	IconUrls struct {
		Small string
		Tiny  string
	}
}
