package output

import (
	"encoding/csv"
	"os"
	"time"

	"github.com/leaklessgfy/safran-server/utils"

	"github.com/leaklessgfy/safran-server/entity"
)

type CSVOutput struct {
	file     *os.File
	writer   *csv.Writer
	date     time.Time
	measures []*entity.Measure
}

func NewCSVOutput(file *os.File) *CSVOutput {
	return &CSVOutput{file: file, writer: csv.NewWriter(file)}
}

func (o *CSVOutput) SaveExperiment(experiment *entity.Experiment) error {
	o.date = experiment.StartDate
	return nil
}

func (o *CSVOutput) SaveMeasures(measures []*entity.Measure) error {
	o.measures = measures
	return nil
}

func (o CSVOutput) SaveSamples(samples []*entity.Sample) error {
	results, err := o.normalizeSamples(samples)
	if err != nil {
		return err
	}
	err = o.writer.WriteAll(results)
	if err != nil {
		return err
	}
	o.writer.Flush()
	return o.writer.Error()
}

func (o CSVOutput) SaveAlarms([]*entity.Alarm) error {
	return nil
}

func (o CSVOutput) Cancel() error {
	err := o.file.Close()
	if err != nil {
		return err
	}
	return os.Remove(o.file.Name())
}

func (o CSVOutput) End() error {
	return o.file.Close()
}

func (o CSVOutput) normalizeSamples(samples []*entity.Sample) ([][]string, error) {
	var results [][]string

	for _, sample := range samples {
		result, err := o.normalizeSample(sample)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

func (o CSVOutput) normalizeSample(sample *entity.Sample) ([]string, error) {
	var result []string
	measure := o.measures[sample.Inc]
	t, err := utils.ParseTime(sample.Time, o.date)

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
