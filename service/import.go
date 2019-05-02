package service

import (
	"io"

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
}

// NewImportService create the import service
func NewImportService(
	influx *InfluxService,
	samplesReader, alarmsReader io.Reader,
	samplesSize, alarmsSize int64,
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
	}, nil
}

// ImportExperiment will import the experiment
func (i *ImportService) ImportExperiment(report *entity.Report, experiment *entity.Experiment) error {
	header, sizeHeader, err := i.samplesParser.ParseHeader()
	report.ReadSamples(sizeHeader)
	if i.handleError(err, report, entity.ReportStepParseHeader) {
		return err
	}

	experiment.StartDate, err = utils.ParseDate(header.StartDate)
	if i.handleError(err, report, entity.ReportStepParseDate) {
		return err
	}

	experiment.EndDate, err = utils.ParseDate(header.EndDate)
	if i.handleError(err, report, entity.ReportStepParseDate) {
		return err
	}

	experiment.ID, err = i.influx.InsertExperiment(*experiment)
	report.ExperimentID = experiment.ID
	if i.handleError(err, report, entity.ReportStepInsertExperiment) {
		i.influx.RemoveExperiment(experiment.ID)
		return err
	}

	return nil
}

// ImportSamples will import measures and samples
func (i *ImportService) ImportSamples(report entity.Report, experiment entity.Experiment, channel chan entity.Report, save chan ChannelRequest) {
	report.Title = "Measures"
	measures, sizeMeasures, err := i.samplesParser.ParseMeasures()
	report.ReadSamples(sizeMeasures)

	if i.handleError(err, &report, entity.ReportStepParseMeasures) {
		channel <- report
		return
	}

	measuresID, err := i.influx.InsertMeasures(experiment.ID, measures)
	if i.handleError(err, &report, entity.ReportStepInsertMeasures) {
		channel <- report
		return
	}

	i.samplesParser.ParseSamples(len(measuresID), func(samples []*entity.Sample, size int, end bool) {
		report.ReadSamples(size)
		if report.HasError() {
			return
		}
		batchPoints, err := i.influx.InsertSamples(experiment.ID, measuresID, experiment.StartDate, samples)
		if err != nil {
			report.AddError(entity.ReportStepInsertSamples, err)
		} else if !report.HasError() && end {
			report.AddSuccess(entity.ReportStepInsertSamples)
			report.Status = entity.ReportStatusSuccess
			report.Progress = 100
			save <- ChannelRequest{report, batchPoints}
		} else {
			report.Step()
			save <- ChannelRequest{report, batchPoints}
		}
		channel <- report
	})
}

// ImportAlarms will import the alarms
func (i *ImportService) ImportAlarms(report entity.Report, experiment entity.Experiment, channel chan entity.Report) {
	if i.alarmsParser == nil {
		return
	}

	report.Title = "Alarms"
	alarms, size, err := i.alarmsParser.ParseAlarms()
	if i.handleError(err, &report, entity.ReportStepParseAlarms) {
		channel <- report
		return
	}

	report.ReadAlarms(size)
	err = i.influx.InsertAlarms(experiment.ID, experiment.StartDate, alarms)
	if i.handleError(err, &report, entity.ReportStepInsertAlarms) {
		channel <- report
		return
	}

	report.Status = entity.ReportStatusSuccess
	report.Progress = 100

	channel <- report
}

func (i *ImportService) handleError(err error, report *entity.Report, step string) bool {
	if err == nil {
		report.AddSuccess(step)
		return false
	}

	report.AddError(step, err)
	if report.ExperimentID != "" {
		if errRemove := i.influx.RemoveExperiment(report.ExperimentID); errRemove != nil {
			report.AddError(entity.ReportStepRemoveExperiment, errRemove)
		} else {
			report.AddSuccess(entity.ReportStepRemoveExperiment)
		}
	}

	return true
}
