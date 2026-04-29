package models

type PlayerItemLevel struct {
	Level              int
	Name               string
	MaxLevel           int
	Village            Village
	SuperTroopIsActive bool
	Equipment          []struct {
		Name     string
		Level    int
		MaxLevel int
		Village  Village
	}
}
