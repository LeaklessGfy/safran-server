package facade

import (
	"context"
	"io"
	"log"
	"strconv"

	"github.com/leaklessgfy/safran-server/entity"
	"github.com/leaklessgfy/safran-server/observer"
	"github.com/leaklessgfy/safran-server/output"
	"github.com/leaklessgfy/safran-server/parser"
	"github.com/leaklessgfy/safran-server/utils"
)

type ParserFacade struct {
	output        output.Output
	observer      observer.Observer
	samplesParser *parser.SamplesParser
	alarmsParser  *parser.AlarmsParser
	ctx           context.Context
	stop          context.CancelFunc
	events        chan Event
}

func NewParserFacade(output output.Output, observer observer.Observer, samplesReader, alarmsReader io.Reader) *ParserFacade {
	samplesParser := parser.NewSamplesParser(samplesReader)
	var alarmsParser *parser.AlarmsParser
	if alarmsReader != nil {
		alarmsParser = parser.NewAlarmsParser(alarmsReader)
	}
	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan Event, 10)

	return &ParserFacade{
		output:        output,
		observer:      observer,
		samplesParser: samplesParser,
		alarmsParser:  alarmsParser,
		ctx:           ctx,
		stop:          cancel,
		events:        events,
	}
}

func (p ParserFacade) Parse(experiment *entity.Experiment) error {
	err := p.importExperiment(experiment)
	if err != nil {
		p.output.Cancel()
		return err
	}
	max := 2
	if p.alarmsParser == nil {
		max = 1
	}
	go p.initEvents(max)
	go p.importFull()
	if p.alarmsParser != nil {
		go p.importAlarms()
	}
	return nil
}

func (p ParserFacade) initEvents(max int) {
	var inc int
	var err error

	for {
		select {
		case <-p.ctx.Done():
			return
		case event := <-p.events:
			switch event.id {
			case EndID:
				inc++
				if inc == max {
					p.stop()
					err = p.output.End()
					if err != nil {
						p.output.Cancel()
						return
					}
					p.observer.OnStep(event.step)
					return
				}
				break
			case MeasureID:
				err = p.output.SaveMeasures(event.measures)
				if p.handleError(event.step, err) {
					return
				}
				break
			case SamplesID:
				err = p.output.SaveSamples(event.samples)
				if p.handleError(event.step, err) {
					return
				}
				break
			case AlarmsID:
				err = p.output.SaveAlarms(event.alarms)
				if p.handleError(event.step, err) {
					return
				}
				break
			}
		}
	}
}

func (p ParserFacade) importExperiment(experiment *entity.Experiment) error {
	header, size, err := p.samplesParser.ParseHeader()
	p.observer.OnRead(size)
	if p.handleError(entity.StepParseHeader, err) {
		return err
	}

	experiment.StartDate, err = utils.ParseDate(header.StartDate)
	if p.handleError(entity.StepParseStartDate, err) {
		return err
	}

	experiment.EndDate, err = utils.ParseDate(header.EndDate)
	if p.handleError(entity.StepParseEndDate, err) {
		return err
	}

	err = p.output.SaveExperiment(experiment)
	if p.handleError(entity.StepSaveExperiment, err) {
		return err
	}

	return nil
}

func (p ParserFacade) importFull() {
	err := p.importMeasures()
	if err != nil || p.hasError() {
		return
	}
	p.importSamples()
}

func (p ParserFacade) importMeasures() error {
	if p.hasError() {
		return nil
	}

	measures, size, err := p.samplesParser.ParseMeasures()
	p.observer.OnRead(size)
	if p.handleError(entity.StepParseMeasures, err) || p.hasError() {
		return err
	}
	p.dispatchMeasures(measures)

	return nil
}

func (p ParserFacade) importSamples() {
	if p.hasError() {
		return
	}

	inc := 0
	for !p.hasError() {
		inc++
		strInc := strconv.Itoa(inc)

		samples, size, end := p.samplesParser.ParseSamples(500)
		p.observer.OnRead(size)
		p.observer.OnStep(entity.StepParseSamples + strInc)

		p.dispatchSamples(samples, strInc)
		if p.hasError() {
			return
		}

		if end {
			p.dispatchEnd()
			p.observer.OnEndSamples()
			return
		}
	}
}

func (p ParserFacade) importAlarms() {
	if p.hasError() {
		return
	}

	alarms, size, err := p.alarmsParser.ParseAlarms()
	p.observer.OnRead(size)
	if p.handleError(entity.StepParseAlarms+"1", err) || p.hasError() {
		return
	}

	p.dispatchAlarms(alarms, "1")
	if p.hasError() {
		return
	}

	p.dispatchEnd()
	p.observer.OnEndAlarms()
}

func (p ParserFacade) dispatchMeasures(measures []*entity.Measure) {
	p.events <- Event{
		id:       MeasureID,
		step:     entity.StepSaveMeasures,
		measures: measures,
	}
}

func (p ParserFacade) dispatchSamples(samples []*entity.Sample, inc string) {
	p.events <- Event{
		id:      SamplesID,
		step:    entity.StepSaveSamples + inc,
		samples: samples,
	}
}

func (p ParserFacade) dispatchAlarms(alarms []*entity.Alarm, inc string) {
	p.events <- Event{
		id:     AlarmsID,
		step:   entity.StepSaveAlarms + inc,
		alarms: alarms,
	}
}

func (p ParserFacade) dispatchEnd() {
	p.events <- Event{id: EndID, step: entity.StepFullEnd}
}

func (p ParserFacade) handleError(step string, err error) bool {
	p.observer.OnStep(step)
	if err != nil {
		p.stop()
		errCancel := p.output.Cancel()
		if errCancel != nil {
			log.Println("[ERROR CANCEL]", errCancel)
		}
		p.observer.OnError(step, err)
		p.observer.OnStep(entity.StepCancel)
		return true
	}
	return false
}

func (p ParserFacade) hasError() bool {
	select {
	case <-p.ctx.Done():
		return true
	default:
		return false
	}
}
