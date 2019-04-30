package parser

import (
	"bufio"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/leaklessgfy/safran-server/entity"
)

type AlarmsParser struct {
	scanner *bufio.Scanner
}

// NewAlarmsParser create a Sample Parser with the scanner
func NewAlarmsParser(reader io.Reader) *AlarmsParser {
	return &AlarmsParser{bufio.NewScanner(reader)}
}

// ParseAlarms parse alarms in the file
func (p AlarmsParser) ParseAlarms() ([]*entity.Alarm, int, error) {
	var alarms []*entity.Alarm
	fullSize := 0
	for p.scanner.Scan() {
		line := p.scanner.Text()
		if len(line) < 1 {
			return alarms, fullSize, nil
		}
		fullSize += len([]byte(line))
		arr := strings.Split(line, separator)
		if len(arr) < 3 {
			return alarms, fullSize, errors.New("Badly formatted alarm line")
		}
		time := strings.Split(arr[0], " ")
		if len(time) < 2 {
			return alarms, fullSize, errors.New("Badly formatted alarm time")
		}
		level, err := strconv.Atoi(arr[1])
		if err != nil {
			return alarms, fullSize, err
		}
		alarms = append(alarms, &entity.Alarm{Time: time[1], Level: level, Message: arr[2]})
	}
	return alarms, fullSize, nil
}
