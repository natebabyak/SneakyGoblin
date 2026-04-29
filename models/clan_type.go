package models

type ClanType string

const (
	OPEN        ClanType = "OPEN"
	INVITE_ONLY ClanType = "INVITE_ONLY"
	CLOSED      ClanType = "CLOSED"
)
