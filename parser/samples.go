package parser

import (
	"bufio"
	"errors"
	"io"
	"strings"

	"github.com/leaklessgfy/safran-server/entity"
)

type SamplesParser struct {
	scanner *bufio.Scanner
}

type Header struct {
	StartDate string
	EndDate   string
}

const offset = 2
const separator = ";"
const nan = "NaN"

// NewSamplesParser create a Sample Parser with the scanner
func NewSamplesParser(reader io.Reader) *SamplesParser {
	return &SamplesParser{bufio.NewScanner(reader)}
}

// ParseHeader parse the start and end date of the file
func (p SamplesParser) ParseHeader() (*Header, int, error) {
	startDate, sizeStart, err := p.parseDate()
	if err != nil {
		return nil, 0, err
	}
	endDate, sizeEnd, err := p.parseDate()
	if err != nil {
		return nil, 0, err
	}
	return &Header{startDate, endDate}, sizeStart + sizeEnd, nil
}

// ParseMeasures parse the measures of the file
func (p SamplesParser) ParseMeasures() ([]*entity.Measure, int, error) {
	measures, sizeM, err := p.parseMeasures()
	if err != nil {
		return nil, 0, err
	}
	types, sizeT, err := parseLine(p.scanner, 2, 0)
	if err != nil {
		return nil, 0, err
	}
	units, sizeU, err := parseLine(p.scanner, 2, 0)
	if err != nil {
		return nil, 0, err
	}
	err = p.mergeTypesUnits(measures, types, units)
	if err != nil {
		return nil, 0, err
	}
	return measures, sizeM + sizeT + sizeU, nil
}

// ParseSamples parse the samples of the file
func (p SamplesParser) ParseSamples(size int, executor func([]*entity.Sample, int, bool)) {
	for true {
		var samples []*entity.Sample
		var size int
		for n := 0; n < 500; n++ {
			if !p.scanner.Scan() {
				executor(samples, size, true)
				return
			}
			line := p.scanner.Text()
			size += len([]byte(line))
			arr := strings.Split(line, separator)
			for i := 2; i < len(arr); i++ {
				if len(arr[i]) > 0 && arr[i] != nan && i < size {
					samples = append(samples, &entity.Sample{Value: arr[i], Time: arr[1], Measure: i - offset})
				}
			}
		}
		executor(samples, size, false)
	}
}

func (p SamplesParser) parseDate() (string, int, error) {
	arr, size, err := parseLine(p.scanner, 1, 1)
	if err != nil {
		return "", 0, err
	}
	if len(arr) < 1 {
		return "", 0, errors.New("")
	}
	return arr[0], size, nil
}

func (p SamplesParser) parseMeasures() ([]*entity.Measure, int, error) {
	arr, size, err := parseLine(p.scanner, 2, 0)
	if err != nil {
		return nil, 0, err
	}
	var measures []*entity.Measure
	for _, m := range arr {
		measures = append(measures, &entity.Measure{Name: m})
	}
	p.scanner.Scan()
	b := []byte(p.scanner.Text())
	return measures, size + len(b), nil
}

func (p SamplesParser) mergeTypesUnits(measures []*entity.Measure, types, units []string) error {
	if len(types) != len(measures) {
		return errors.New("Types length != measures length")
	}
	if len(units) != len(measures) {
		return errors.New("Units length != measures length")
	}
	for i, typex := range types {
		measures[i].Typex = typex
	}
	for i, unitx := range units {
		measures[i].Unitx = unitx
	}
	return nil
}
