package facade

import (
	"context"
	"io"
	"strconv"

	"github.com/leaklessgfy/safran-server/entity"
	"github.com/leaklessgfy/safran-server/observer"
	"github.com/leaklessgfy/safran-server/parser"
	"github.com/leaklessgfy/safran-server/saver"
	"github.com/leaklessgfy/safran-server/utils"
)

type ParserFacade struct {
	saver         saver.Saver
	observer      observer.Observer
	samplesParser *parser.SamplesParser
	alarmsParser  *parser.AlarmsParser
	ctx           context.Context
	cancel        context.CancelFunc
	events        chan Event
}

func NewParserFacade(saver saver.Saver, observer observer.Observer, samplesReader, alarmsReader io.Reader) *ParserFacade {
	samplesParser := parser.NewSamplesParser(samplesReader)
	var alarmsParser *parser.AlarmsParser
	if alarmsReader != nil {
		alarmsParser = parser.NewAlarmsParser(alarmsReader)
	}
	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan Event, 10)

	return &ParserFacade{
		saver:         saver,
		observer:      observer,
		samplesParser: samplesParser,
		alarmsParser:  alarmsParser,
		ctx:           ctx,
		cancel:        cancel,
		events:        events,
	}
}

func (p ParserFacade) Parse(experiment *entity.Experiment) error {
	err := p.importExperiment(experiment)
	if err != nil {
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
	handleError := func(step string, err error) bool {
		p.observer.OnStep(step)
		if err != nil {
			p.cancel()
			p.observer.OnError(step, err)
			p.saver.Cancel()
			p.observer.OnStep(entity.StepCancel)
			return true
		}
		return false
	}

	for {
		select {
		case <-p.ctx.Done():
			p.saver.Cancel()
			p.observer.OnStep(entity.StepCancel)
			return
		case event := <-p.events:
			switch event.id {
			case EndID:
				inc++
				if inc == max {
					p.observer.OnStep(event.step)
					p.cancel()
					return
				}
				break
			case MeasureID:
				err = p.saver.SaveMeasures(event.measures)
				if handleError(event.step, err) {
					return
				}
				break
			case SamplesID:
				err = p.saver.SaveSamples(event.samples)
				if handleError(event.step, err) {
					return
				}
				break
			case AlarmsID:
				err = p.saver.SaveAlarms(event.alarms)
				if handleError(event.step, err) {
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
	p.observer.OnStep(entity.StepParseHeader)
	if err != nil {
		p.observer.OnError(entity.StepParseHeader, err)
		return err
	}

	experiment.StartDate, err = utils.ParseDate(header.StartDate)
	p.observer.OnStep(entity.StepParseStartDate)
	if err != nil {
		p.observer.OnError(entity.StepParseStartDate, err)
		return err
	}

	experiment.EndDate, err = utils.ParseDate(header.EndDate)
	p.observer.OnStep(entity.StepParseEndDate)
	if err != nil {
		p.observer.OnError(entity.StepParseEndDate, err)
		return err
	}

	err = p.saver.SaveExperiment(experiment)
	p.observer.OnStep(entity.StepSaveExperiment)
	if err != nil {
		p.observer.OnError(entity.StepSaveExperiment, err)
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
	p.observer.OnStep(entity.StepParseMeasures)
	if err != nil || p.hasError() {
		p.observer.OnError(entity.StepParseMeasures, err)
		p.cancel()
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
	p.observer.OnStep(entity.StepParseAlarms + "1")
	if err != nil || p.hasError() {
		p.observer.OnError(entity.StepParseAlarms+"1", err)
		p.cancel()
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

func (p ParserFacade) hasError() bool {
	select {
	case <-p.ctx.Done():
		return true
	default:
		return false
	}
}
