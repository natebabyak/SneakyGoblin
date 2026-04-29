package models

type Role string

const (
	NOT_MEMBER Role = "NOT_MEMBER"
	MEMBER     Role = "MEMBER"
	LEADER     Role = "LEADER"
	ADMIN      Role = "ADMIN"
	COLEADER   Role = "COLEADER"
)
