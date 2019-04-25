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
func (p AlarmsParser) ParseAlarms() ([]*entity.Alarm, error) {
	var alarms []*entity.Alarm
	for p.scanner.Scan() {
		line := p.scanner.Text()
		if len(line) < 1 {
			return alarms, nil
		}
		arr := strings.Split(line, separator)
		if len(arr) < 3 {
			return alarms, errors.New("")
		}
		time := arr[0]
		level, err := strconv.Atoi(arr[1])
		if err != nil {
			return alarms, err
		}
		alarms = append(alarms, &entity.Alarm{Time: time, Level: level, Message: arr[2]})
	}
	return alarms, nil
}
