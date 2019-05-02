package entity

import (
	"errors"
	"time"
)

// Experiment is the experiment
type Experiment struct {
	ID        string
	Reference string
	Name      string
	Bench     string
	Campaign  string
	StartDate time.Time
	EndDate   time.Time
}

// Validate check if current experiement is valide
func (e Experiment) Validate() error {
	if e.Reference == "" {
		return errors.New("experiment reference should not be null")
	}

	if e.Name == "" {
		return errors.New("experiment name should not be null")
	}

	if e.Bench == "" {
		return errors.New("experiment bench should not be null")
	}

	if e.Campaign == "" {
		return errors.New("experiment campaign should not be null")
	}

	return nil
}
