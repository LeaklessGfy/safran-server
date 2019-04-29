package service

import (
	"mime/multipart"
	"net/http"
)

type Sizer interface {
	Size() int64
}

func ExtractSamples(r *http.Request) (multipart.File, int64, error) {
	samplesFile, _, err := r.FormFile("samples")
	if err != nil {
		return nil, 0, err
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
	return file.(Sizer).Size(), nil
}
