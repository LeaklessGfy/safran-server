package entity

import "time"

type Experiment struct {
	ID        string
	Reference string
	Name      string
	Bench     string
	Campaign  string
	StartDate time.Time
	EndDate   time.Time
}
