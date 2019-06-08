package output

import (
	"errors"
	"os"
)

func NewOutput(key string) (Output, error) {
	switch key {
	case "csv":
		file, err := os.Create("./csv/result.csv")
		if err != nil {
			return nil, err
		}
		return NewCSVOutput(file), nil
	case "json":
		return &JSONOutput{}, nil
	case "influx":
		return NewInfluxOutput()
	case "fake":
		return &EmptyOutput{}, nil
	}
	return nil, errors.New("no output associated with " + key)
}
