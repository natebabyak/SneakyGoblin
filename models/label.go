package models

type Label struct {
	name     string
	id       int
	iconUrls struct {
		small  string
		medium string
	}
}
