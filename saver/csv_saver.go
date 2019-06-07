package saver

import "github.com/leaklessgfy/safran-server/entity"

type CSVSaver struct{}

func (s CSVSaver) SaveExperiment(*entity.Experiment) error {
	return nil
}

func (s CSVSaver) SaveMeasures([]*entity.Measure) error {
	return nil
}

func (s CSVSaver) SaveSamples([]*entity.Sample) error {
	return nil
}

func (s CSVSaver) SaveAlarms([]*entity.Alarm) error {
	return nil
}

func (s CSVSaver) Cancel() error {
	return nil
}
