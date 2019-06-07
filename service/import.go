package service

import (
	"context"
	"io"
	"strconv"

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
	reqSignal     chan ClientRequest
	ctx           context.Context
	cancel        context.CancelFunc
}

type ClientRequest struct {
	step        string
	stop        bool
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

	ctx, cancel := context.WithCancel(context.Background())

	return &ImportService{
		influx:        influx,
		samplesParser: samplesParser,
		alarmsParser:  alarmsParser,
		reqSignal:     make(chan ClientRequest, 50),
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

// ImportExperiment will import the experiment
func (i ImportService) ImportExperiment(report *entity.Report, experiment *entity.Experiment) error {
	header, sizeHeader, err := i.samplesParser.ParseHeader()
	report.AddRead(sizeHeader)
	if i.handleError(err, report, entity.ReportStepParseHeader) {
		return err
	}

	experiment.Date, err = utils.ParseDate(header.StartDate)
	if i.handleError(err, report, entity.ReportStepParseDate) {
		return err
	}

	// Parse time to timestamp?
	experiment.StartTime = header.StartDate
	experiment.EndTime = header.EndDate
	experiment.ID, err = i.influx.InsertExperiment(*experiment)
	report.ExperimentID = experiment.ID
	if i.handleError(err, report, entity.ReportStepInsertExperiment) {
		return err
	}

	return nil
}

// ImportSamples will import measures and samples
func (i ImportService) ImportSamples(report entity.Report, experiment entity.Experiment, channel chan entity.Report) {
	measures, sizeMeasures, err := i.samplesParser.ParseMeasures()
	report.AddRead(sizeMeasures)

	if i.handleErrorChan(err, &report, entity.ReportStepParseMeasures, channel) || i.hasError() {
		return
	}

	measuresID, err := i.influx.InsertMeasures(experiment.ID, measures)
	report.Step()

	if i.handleErrorChan(err, &report, entity.ReportStepInsertMeasures, channel) {
		return
	}

	inc := 0
	for !i.hasError() {
		inc++
		samples, sizeSamples, end := i.samplesParser.ParseSamples(500, len(measuresID))
		report.AddRead(sizeSamples).Step()

		batchPoints, err := i.influx.PrepareSamples(experiment.ID, measuresID, experiment.Date, samples)
		if i.handleErrorChan(err, &report, entity.ReportStepPrepareSamples+strconv.Itoa(inc), channel) || i.hasError() {
			return
		}

		i.reqSignal <- ClientRequest{
			step:        entity.ReportStepInsertSamples + strconv.Itoa(inc),
			batchPoints: batchPoints,
		}

		if end {
			report.End()
			channel <- report
			i.reqSignal <- ClientRequest{stop: true}
			return
		}

		channel <- report
	}
}

// ImportAlarms will import the alarms
func (i ImportService) ImportAlarms(report entity.Report, experiment entity.Experiment, channel chan entity.Report) {
	if i.alarmsParser == nil || i.hasError() {
		return
	}

	alarms, size, err := i.alarmsParser.ParseAlarms()
	report.AddRead(size)

	if i.handleErrorChan(err, &report, entity.ReportStepParseAlarms, channel) || i.hasError() {
		return
	}

	batchPoints, err := i.influx.PrepareAlarms(experiment.ID, experiment.Date, alarms)
	report.Step()

	if i.handleErrorChan(err, &report, entity.ReportStepPrepareAlarms, channel) || i.hasError() {
		return
	}

	report.End()
	channel <- report
	i.reqSignal <- ClientRequest{
		step:        entity.ReportStepInsertAlarms,
		batchPoints: batchPoints,
	}
	i.reqSignal <- ClientRequest{stop: true}
}

func (i ImportService) Save(report entity.Report, channel chan entity.Report) {
	inc := 0
	for {
		select {
		case <-i.ctx.Done():
			return
		case request := <-i.reqSignal:
			if request.stop {
				inc++
				if inc == 2 {
					report.End()
					channel <- report
					return
				}
				break
			}
			err := i.influx.InsertBatchPoints(request.batchPoints)
			report.Step()
			if i.handleErrorChan(err, &report, request.step, channel) {
				return
			}
		}
	}
}

func (i ImportService) handleErrorChan(err error, report *entity.Report, step string, channel chan entity.Report) bool {
	if err == nil {
		report.AddSuccess(step)
		channel <- *report
		return false
	}

	i.cancel()
	report.AddError(step, err)
	i.removeExperiment(report)
	channel <- *report

	return true
}

func (i ImportService) handleError(err error, report *entity.Report, step string) bool {
	if err == nil {
		report.AddSuccess(step)
		return false
	}

	i.cancel()
	report.AddError(step, err)
	i.removeExperiment(report)

	return true
}

func (i ImportService) removeExperiment(report *entity.Report) {
	if report.ExperimentID != "" {
		if errRemove := i.influx.RemoveExperiment(report.ExperimentID); errRemove != nil {
			report.AddError(entity.ReportStepRemoveExperiment, errRemove)
		} else {
			report.AddSuccess(entity.ReportStepRemoveExperiment)
		}
	}
}

func (i ImportService) hasError() bool {
	select {
	case <-i.ctx.Done():
		return true
	default:
		return false
	}
}
