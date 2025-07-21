package model

import "time"

type Post struct {
	ID         int64
	Link       string
	Text       string
	Timestamp  time.Time
	Username   string
	Regions    []string
	ErrandType bool
	ErrorType  string
}
