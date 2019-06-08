package output

import (
	"github.com/leaklessgfy/safran-server/entity"
)

type Output interface {
	SaveExperiment(*entity.Experiment) error
	SaveMeasures([]*entity.Measure) error
	SaveSamples([]*entity.Sample) error
	SaveAlarms([]*entity.Alarm) error
	Cancel() error
	End() error
}
