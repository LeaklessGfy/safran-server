package service

import (
	"io"
	"strconv"
	"sync"

	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/leaklessgfy/safran-server/entity"
	"github.com/leaklessgfy/safran-server/utils"

	"github.com/leaklessgfy/safran-server/parser"
)

// ImportService is the service use to orchestrate parsing and inserting inside influx
type ImportService struct {
	influx        *InfluxService
	samplesParser *parser.SamplesParser
	alarmsParser  *parser.AlarmsParser
	channel       chan ClientRequest
	errors        chan bool
	stop          bool
	lock          sync.Mutex
}

type ClientRequest struct {
	step        string
	batchPoints client.BatchPoints
}

// NewImportService create the import service
func NewImportService(
	influx *InfluxService,
	samplesReader, alarmsReader io.Reader,
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
		channel:       make(chan ClientRequest, 50),
		errors:        make(chan bool, 2),
	}, nil
}

// ImportExperiment will import the experiment
func (i *ImportService) ImportExperiment(report *entity.Report, experiment *entity.Experiment) error {
	header, sizeHeader, err := i.samplesParser.ParseHeader()
	report.AddRead(sizeHeader)
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
func (i *ImportService) ImportSamples(report entity.Report, experiment entity.Experiment, channel chan entity.Report) {
	measures, sizeMeasures, err := i.samplesParser.ParseMeasures()
	report.AddRead(sizeMeasures)

	if i.handleError(err, &report, entity.ReportStepParseMeasures) {
		channel <- report
		return
	}
	channel <- report

	if i.hasError() {
		return
	}

	measuresID, err := i.influx.InsertMeasures(experiment.ID, measures)
	report.Step()

	if i.handleError(err, &report, entity.ReportStepInsertMeasures) {
		channel <- report
		return
	}
	channel <- report

	inc := 0
	for !i.hasError() {
		inc++
		samples, sizeSamples, end := i.samplesParser.ParseSamples(500, len(measuresID))
		report.AddRead(sizeSamples).Step()

		batchPoints, err := i.influx.PrepareSamples(experiment.ID, measuresID, experiment.StartDate, samples)
		if i.handleError(err, &report, entity.ReportStepPrepareSamples+strconv.Itoa(inc)) {
			channel <- report
			return
		}

		if end {
			report.End()
		}

		if i.hasError() {
			return
		}

		channel <- report
		i.channel <- ClientRequest{entity.ReportStepInsertSamples + strconv.Itoa(inc), batchPoints}

		if end {
			i.channel <- ClientRequest{step: "1"}
			return
		}
	}
}

// ImportAlarms will import the alarms
func (i *ImportService) ImportAlarms(report entity.Report, experiment entity.Experiment, channel chan entity.Report) {
	if i.alarmsParser == nil {
		return
	}

	alarms, size, err := i.alarmsParser.ParseAlarms()
	report.AddRead(size)

	if i.handleError(err, &report, entity.ReportStepParseAlarms) {
		channel <- report
		return
	}

	batchPoints, err := i.influx.PrepareAlarms(experiment.ID, experiment.StartDate, alarms)
	report.Step()

	if i.handleError(err, &report, entity.ReportStepPrepareAlarms) {
		channel <- report
		return
	}

	report.End()

	if i.hasError() {
		return
	}

	channel <- report
	i.channel <- ClientRequest{entity.ReportStepInsertAlarms, batchPoints}
	i.channel <- ClientRequest{step: "1"}
}

func (i *ImportService) Save(report entity.Report, channel chan entity.Report) {
	inc := 0
	for {
		select {
		case <-i.errors:
			return
		case request := <-i.channel:
			if request.step == "1" {
				inc++
				if inc == 2 {
					report.End()
					channel <- report
				}
				break
			}

			err := i.influx.InsertBatchPoints(request.batchPoints)
			report.Step()

			if i.handleError(err, &report, request.step) {
				channel <- report
				return
			}

			channel <- report
		}
	}
}

func (i *ImportService) handleError(err error, report *entity.Report, step string) bool {
	if err == nil {
		report.AddSuccess(step)
		return false
	}

	i.lock.Lock()
	i.stop = true
	i.lock.Unlock()

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

func (i *ImportService) hasError() bool {
	i.lock.Lock()
	defer i.lock.Unlock()
	return i.stop
}
