package models

type PlayerHouse struct {
	Elements []struct {
		Type PlayerHouseElementType
		Id   int
	}
}
