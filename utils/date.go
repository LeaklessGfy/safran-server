package utils

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// ParseDate parse a date representation (ex: ) to Time struct
func ParseDate(str string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05.000", str)
}

// ParseTime parse a time representation (ex: 12:03:00 or 12:04:05.555) to Time struct
func ParseTime(str string, date time.Time) (time.Time, error) {
	hour := date.Hour()
	min := date.Minute()
	sec := date.Second()
	nano := date.Nanosecond()

	strSplit := strings.FieldsFunc(str, func(r rune) bool {
		return r == ':' || r == '.' || r == ','
	})

	if len(strSplit) == 3 {
		min, _ = strconv.Atoi(strSplit[0])
		sec, _ = strconv.Atoi(strSplit[1])
		nano, _ = strconv.Atoi(strSplit[2])
	} else if len(strSplit) == 4 {
		hour, _ = strconv.Atoi(strSplit[0])
		min, _ = strconv.Atoi(strSplit[1])
		sec, _ = strconv.Atoi(strSplit[2])
		nano, _ = strconv.Atoi(strSplit[3])
	} else {
		return date, errors.New("Bad time formatting " + str)
	}

	return time.Date(date.Year(), date.Month(), date.Day(), hour, min, sec, nano, date.Location()), nil
}
