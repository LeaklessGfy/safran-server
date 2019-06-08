package saver

import "github.com/leaklessgfy/safran-server/entity"

type FakeSaver struct{}

func (s FakeSaver) SaveExperiment(*entity.Experiment) error {
	return nil
}

func (s FakeSaver) SaveMeasures([]*entity.Measure) error {
	return nil
}

func (s FakeSaver) SaveSamples([]*entity.Sample) error {
	return nil
}

func (s FakeSaver) SaveAlarms([]*entity.Alarm) error {
	return nil
}

func (s FakeSaver) Cancel() error {
	return nil
}

func (s FakeSaver) End() error {
	return nil
}
