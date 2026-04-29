package models

type PlayerHouse struct {
	elements []struct {
		Type PlayerHouseElementType
		id   int
	}
}
