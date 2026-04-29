package models

type PlayerAchievementProgress struct {
	Stars          int
	Value          int
	Name           string
	Target         int
	Info           string
	CompletionInfo string
	Village        Village
}
