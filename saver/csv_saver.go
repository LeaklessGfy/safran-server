package saver

import (
	"encoding/csv"
	"os"
	"time"

	"github.com/leaklessgfy/safran-server/utils"

	"github.com/leaklessgfy/safran-server/entity"
)

type CSVSaver struct {
	file     *os.File
	writer   *csv.Writer
	date     time.Time
	measures []*entity.Measure
}

func NewCSVSaver(file *os.File) *CSVSaver {
	return &CSVSaver{file: file, writer: csv.NewWriter(file)}
}

func (s *CSVSaver) SaveExperiment(experiment *entity.Experiment) error {
	s.date = experiment.StartDate
	return nil
}

func (s *CSVSaver) SaveMeasures(measures []*entity.Measure) error {
	s.measures = measures
	return nil
}

func (s CSVSaver) SaveSamples(samples []*entity.Sample) error {
	results, err := s.normalizeSamples(samples)
	if err != nil {
		return err
	}
	err = s.writer.WriteAll(results)
	if err != nil {
		return err
	}
	s.writer.Flush()
	return s.writer.Error()
}

func (s CSVSaver) SaveAlarms([]*entity.Alarm) error {
	return nil
}

func (s CSVSaver) Cancel() error {
	err := s.file.Close()
	if err != nil {
		return err
	}
	return os.Remove(s.file.Name())
}

func (s CSVSaver) End() error {
	return s.file.Close()
}

func (s CSVSaver) normalizeSamples(samples []*entity.Sample) ([][]string, error) {
	var results [][]string

	for _, sample := range samples {
		result, err := s.normalizeSample(sample)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

func (s CSVSaver) normalizeSample(sample *entity.Sample) ([]string, error) {
	var result []string
	measure := s.measures[sample.Inc]
	t, err := utils.ParseTime(sample.Time, s.date)

	if err != nil {
		return nil, err
	}

	result = append(result, t.Format(time.RFC3339Nano)) // Bad format
	result = append(result, measure.Name)
	result = append(result, measure.Typex)
	result = append(result, measure.Unitx)
	result = append(result, sample.Value)

	return result, nil
}
