package service

import (
	"errors"
	"io"
	"log"

	"github.com/leaklessgfy/safran-server/entity"
	"github.com/leaklessgfy/safran-server/utils"

	"github.com/leaklessgfy/safran-server/parser"
)

// ImportService is the service use to orchestrate parsing and inserting inside influx
type ImportService struct {
	influx *InfluxService
}

// NewImportService create the import service
func NewImportService() (*ImportService, error) {
	influx, err := NewInfluxService()
	if err != nil {
		return nil, err
	}
	return &ImportService{influx}, nil
}

// ImportExperiment will import the experiment
func (i ImportService) ImportExperiment(experiment entity.Experiment, samplesReader io.Reader) (entity.Experiment, error) {
	samplesParser := parser.NewSamplesParser(samplesReader)

	// parse metadata
	header, err := samplesParser.ParseHeader()
	if err != nil {
		return experiment, errors.New("{Parse Header} - " + err.Error())
	}
	experiment.StartDate, err = utils.ParseDate(header.StartDate)
	if err != nil {
		return experiment, errors.New("{Parse Experiment StartDate} - " + err.Error())
	}
	experiment.EndDate, err = utils.ParseDate(header.EndDate)
	if err != nil {
		return experiment, errors.New("{Parse Experiment EndDate} - " + err.Error())
	}
	experiment.ID, err = i.influx.InsertExperiment(experiment)
	if err != nil {
		return experiment, errors.New("{Insert Experiment} - " + err.Error())
	}

	// parse measures
	measures, err := samplesParser.ParseMeasures()
	if err != nil {
		i.influx.RemoveExperiment(experiment.ID)
		return experiment, errors.New("{Parse Measures} - " + err.Error())
	}
	measuresID, err := i.influx.InsertMeasures(experiment.ID, measures)
	if err != nil {
		i.influx.RemoveExperiment(experiment.ID)
		return experiment, errors.New("{Insert Measures} - " + err.Error())
	}

	// parse samples
	samplesParser.ParseSamples(len(measures), func(samples []*entity.Sample) {
		err := i.influx.InsertSamples(experiment.ID, measuresID, experiment.StartDate, samples)
		if err != nil {
			//i.influx.RemoveExperiment(experimentID)
			log.Println(err)
		}
	})

	return experiment, nil
}

// ImportAlarms will import the alarms
func (i ImportService) ImportAlarms(experiment entity.Experiment, alarmsReader io.Reader) error {
	if alarmsReader == nil {
		return nil
	}
	alarmsParser := parser.NewAlarmsParser(alarmsReader)
	alarms, err := alarmsParser.ParseAlarms()
	if err != nil {
		i.influx.RemoveExperiment(experiment.ID)
		return errors.New("{Parse Alarms} - " + err.Error())
	}
	err = i.influx.InsertAlarms(experiment.ID, experiment.StartDate, alarms)
	if err != nil {
		i.influx.RemoveExperiment(experiment.ID)
		return errors.New("{Insert Alarms} - " + err.Error())
	}
	return nil
}
