package models

type WarFrequency string

const (
	UNKNOWN                 WarFrequency = "UNKNOWN"
	ALWAYS                  WarFrequency = "ALWAYS"
	MORE_THAN_ONCE_PER_WEEK WarFrequency = "MORE_THAN_ONCE_PER_WEEK"
	ONCE_PER_WEEK           WarFrequency = "ONCE_PER_WEEK"
	LESS_THAN_ONCE_PER_WEEK WarFrequency = "LESS_THAN_ONCE_PER_WEEK"
	NEVER                   WarFrequency = "NEVER"
	ANY                     WarFrequency = "ANY"
)
