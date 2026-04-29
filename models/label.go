package models

type Label struct {
	Name     string
	Id       int
	IconUrls struct {
		Small  string
		Medium string
	}
}
