package models

type PlayerAchievementProgress struct {
	stars          int
	value          int
	name           string
	target         int
	info           string
	completionInfo string
	village        Village
}
