package utils

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// ParseDate parse a date representation (ex: ) to Time struct
func ParseDate(str string) (time.Time, error) {
	t, err := time.Parse("2006-01-02T15:04:05.000", str)
	if err != nil {
		return t, err
	}
	return t.UTC(), nil
}

// ParseTime parse a time representation (ex: 12:03:00 or 12:04:05.555) to Time struct
func ParseTime(str string, date time.Time) (time.Time, error) {
	var err error
	hour := date.Hour()
	min := date.Minute()
	sec := date.Second()
	nano := date.Nanosecond()

	strSplit := strings.FieldsFunc(str, func(r rune) bool {
		return r == ':' || r == '.' || r == ','
	})

	if len(strSplit) == 3 {
		min, err = strconv.Atoi(strSplit[0])
		if err != nil {
			return date, err
		}
		sec, err = strconv.Atoi(strSplit[1])
		if err != nil {
			return date, err
		}
		nano, err = strconv.Atoi(strSplit[2])
		if err != nil {
			return date, err
		}
	} else if len(strSplit) == 4 {
		hour, err = strconv.Atoi(strSplit[0])
		if err != nil {
			return date, err
		}
		min, err = strconv.Atoi(strSplit[1])
		if err != nil {
			return date, err
		}
		sec, err = strconv.Atoi(strSplit[2])
		if err != nil {
			return date, err
		}
		nano, err = strconv.Atoi(strSplit[3])
		if err != nil {
			return date, err
		}
	} else {
		return date, errors.New("Bad time formatting " + str)
	}

	return time.Date(date.Year(), date.Month(), date.Day(), hour, min, sec, nano, time.UTC).UTC(), nil
}
