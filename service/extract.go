package service

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/leaklessgfy/safran-server/entity"
	"github.com/leaklessgfy/safran-server/output"
)

type Sizer interface {
	Size() int64
}

func ExtractExperiment(r *http.Request) (*entity.Experiment, error) {
	experimentValue := r.FormValue("experiment")
	if experimentValue == "" {
		return nil, errors.New("experiment info is required")
	}
	var experiment entity.Experiment
	err := json.Unmarshal([]byte(experimentValue), &experiment)
	if err != nil {
		return nil, err
	}
	err = experiment.Validate()
	if err != nil {
		return nil, err
	}
	return &experiment, nil
}

func ExtractOutput(r *http.Request) (output.Output, error) {
	key := r.FormValue("output")
	if key == "" {
		return nil, errors.New("output info is required")
	}
	return output.NewOutput(key)
}

func ExtractSamples(r *http.Request) (multipart.File, int64, error) {
	samplesFile, _, err := r.FormFile("samples")
	if err != nil {
		return nil, 0, errors.New("samples is required " + err.Error())
	}
	samplesSize, err := getSize(samplesFile)
	if err != nil {
		return nil, 0, err
	}
	return samplesFile, samplesSize, nil
}

func ExtractAlarms(r *http.Request) (multipart.File, int64, error) {
	alarmsFile, _, err := r.FormFile("alarms")
	if alarmsFile == nil || err != nil {
		return nil, 0, nil
	}
	alarmsSize, err := getSize(alarmsFile)
	if err != nil {
		return nil, 0, err
	}
	return alarmsFile, alarmsSize, nil
}

func getSize(file multipart.File) (int64, error) {
	fileHeader := make([]byte, 512)
	if _, err := file.Read(fileHeader); err != nil {
		return 0, err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return 0, err
	}
	sz, ok := file.(Sizer)
	if ok {
		return sz.Size(), nil
	}
	fi, ok := file.(*os.File)
	if !ok {
		return 0, errors.New("Can't determine file")
	}
	stats, err := fi.Stat()
	if err != nil {
		return 0, nil
	}
	return stats.Size(), nil
}
