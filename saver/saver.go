package saver

import (
	"github.com/leaklessgfy/safran-server/entity"
)

type Saver interface {
	SaveExperiment(*entity.Experiment) error
	SaveMeasures([]*entity.Measure) error
	SaveSamples([]*entity.Sample) error
	SaveAlarms([]*entity.Alarm) error
	Cancel() error
	End() error
}
