package output

import "github.com/leaklessgfy/safran-server/entity"

type EmptyOutput struct{}

func (o EmptyOutput) SaveExperiment(*entity.Experiment) error {
	return nil
}

func (o EmptyOutput) SaveMeasures([]*entity.Measure) error {
	return nil
}

func (o EmptyOutput) SaveSamples([]*entity.Sample) error {
	return nil
}

func (o EmptyOutput) SaveAlarms([]*entity.Alarm) error {
	return nil
}

func (o EmptyOutput) Cancel() error {
	return nil
}

func (o EmptyOutput) End() error {
	return nil
}
