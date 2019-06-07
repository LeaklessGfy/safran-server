package facade

import "github.com/leaklessgfy/safran-server/entity"

const (
	MeasureID = 1
	SamplesID = 2
	AlarmsID  = 3
	EndID     = 4
)

type Event struct {
	id       int
	step     string
	measures []*entity.Measure
	samples  []*entity.Sample
	alarms   []*entity.Alarm
}
