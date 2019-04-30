package service

import (
	"errors"
	"io"
	"sync"

	"github.com/leaklessgfy/safran-server/entity"
	"github.com/leaklessgfy/safran-server/utils"

	"github.com/leaklessgfy/safran-server/parser"
)

// ImportService is the service use to orchestrate parsing and inserting inside influx
type ImportService struct {
	influx        *InfluxService
	samplesParser *parser.SamplesParser
	alarmsParser  *parser.AlarmsParser
	samplesSize   int64
	alarmsSize    int64
	reports       chan entity.Report
	readSize      int64
	lock          sync.Mutex
}

// NewImportService create the import service
func NewImportService(
	influx *InfluxService,
	samplesReader, alarmsReader io.Reader,
	samplesSize, alarmsSize int64,
	reports chan entity.Report,
) (*ImportService, error) {
	if err := influx.Ping(); err != nil {
		return nil, err
	}

	samplesParser := parser.NewSamplesParser(samplesReader)
	var alarmsParser *parser.AlarmsParser
	if alarmsReader != nil {
		alarmsParser = parser.NewAlarmsParser(alarmsReader)
	}

	return &ImportService{
		influx:        influx,
		samplesParser: samplesParser,
		alarmsParser:  alarmsParser,
		samplesSize:   samplesSize,
		alarmsSize:    alarmsSize,
		reports:       reports,
	}, nil
}

// ImportExperiment will import the experiment
func (i *ImportService) ImportExperiment(report *entity.Report, experiment *entity.Experiment) error {
	header, sizeHeader, err := i.samplesParser.ParseHeader()
	if err != nil {
		return errors.New("Parse Header - " + err.Error())
	}
	experiment.StartDate, err = utils.ParseDate(header.StartDate)
	if err != nil {
		return errors.New("Parse Experiment StartDate - " + err.Error())
	}
	experiment.EndDate, err = utils.ParseDate(header.EndDate)
	if err != nil {
		return errors.New("Parse Experiment EndDate - " + err.Error())
	}
	experiment.ID, err = i.influx.InsertExperiment(*experiment)
	if err != nil {
		i.influx.RemoveExperiment(experiment.ID)
		return errors.New("Insert Experiment - " + err.Error())
	}

	report.ExperimentID = experiment.ID
	report.Progress = i.addSize(sizeHeader)

	return nil
}

func (i ImportService) ImportSamples(report entity.Report, experiment entity.Experiment) {
	measures, sizeMeasures, err := i.samplesParser.ParseMeasures()
	report.Progress = i.addSize(sizeMeasures)

	if err != nil {
		report.AddError(err)
		if errRemove := i.influx.RemoveExperiment(experiment.ID); errRemove != nil {
			report.AddError(errRemove)
		}
		i.reports <- report
		return
	}

	measuresID, err := i.influx.InsertMeasures(experiment.ID, measures)
	if err != nil {
		report.AddError(err)
		if errRemove := i.influx.RemoveExperiment(experiment.ID); errRemove != nil {
			report.AddError(errRemove)
		}
		i.reports <- report
		return
	}

	i.samplesParser.ParseSamples(len(measuresID), func(samples []*entity.Sample, size int, end bool) {
		report.Progress = i.addSize(size)

		err := i.influx.InsertSamples(experiment.ID, measuresID, experiment.StartDate, samples)
		if err != nil {
			//i.influx.RemoveExperiment(experimentID)
			report.AddError(err)
		}
		if len(report.Errors) < 1 && end {
			report.Status = entity.StatusSuccess
			report.Progress = 100
		}
		i.reports <- report
	})
}

// ImportAlarms will import the alarms
func (i *ImportService) ImportAlarms(report entity.Report, experiment entity.Experiment) {
	if i.alarmsParser == nil {
		return
	}

	alarms, size, err := i.alarmsParser.ParseAlarms()
	report.Progress = i.addSize(size)

	if err != nil {
		report.AddError(err)
		if errRemove := i.influx.RemoveExperiment(experiment.ID); errRemove != nil {
			report.AddError(errRemove)
		}
		i.reports <- report
		return
	}

	err = i.influx.InsertAlarms(experiment.ID, experiment.StartDate, alarms)
	if err != nil {
		report.AddError(err)
		if errRemove := i.influx.RemoveExperiment(experiment.ID); errRemove != nil {
			report.AddError(errRemove)
		}
		i.reports <- report
		return
	}

	i.reports <- report
}

func (i *ImportService) addSize(size int) int {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.readSize += int64(size)
	return int((i.readSize * 100) / (i.samplesSize + i.alarmsSize))
}
