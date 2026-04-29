package models

type PlayerItemLevel struct {
	level              int
	name               string
	maxLevel           int
	village            Village
	superTroopIsActive bool
	equipment          []struct {
		name     string
		level    int
		maxLevel int
		village  Village
	}
}
