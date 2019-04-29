package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/leaklessgfy/safran-server/entity"

	"github.com/leaklessgfy/safran-server/service"
)

// Server is an abstraction layer for http server
type Server struct {
	reports chan entity.Report
	influx  *service.InfluxService
}

// NewServer create a server instance
func NewServer() (*Server, error) {
	influx, err := service.NewInfluxService()

	if err != nil {
		return nil, err
	}

	return &Server{
		reports: make(chan entity.Report),
		influx:  influx,
	}, nil
}

// Start will start the http server and setup routes
func (s Server) Start() error {
	http.HandleFunc("/simple", s.simpleHandler)
	http.HandleFunc("/upload", s.uploadHandler)
	http.HandleFunc("/events", s.eventsHandler)
	log.Println("Server Start on :8888")

	return http.ListenAndServe(":8888", nil)
}

func (s Server) simpleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	s.reports <- entity.NewReport()

	fmt.Fprintf(w, "toto\n")
}

func (s Server) uploadHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	r.ParseMultipartForm(32 << 20)

	jsonR := json.NewEncoder(w)
	report := entity.NewReport()

	// EXPERIMENT
	experimentValue := r.FormValue("experiment")
	if experimentValue == "" {
		report.AddError(errors.New("experiment info is required"))
		jsonR.Encode(report)
		return
	}
	var experiment entity.Experiment
	err := json.Unmarshal([]byte(experimentValue), &experiment)
	if err != nil {
		jsonR.Encode(report.AddError(err))
		return
	}
	err = experiment.Validate()
	if err != nil {
		jsonR.Encode(report.AddError(err))
		return
	}

	// FILES
	samplesFile, samplesSize, err := service.ExtractSamples(r)
	if err != nil {
		jsonR.Encode(report.AddError(err))
		return
	}
	defer samplesFile.Close()

	alarmsFile, _, err := service.ExtractAlarms(r)
	if err != nil {
		jsonR.Encode(report.AddError(err))
		return
	}
	if alarmsFile != nil {
		report.HasAlarms = true
		defer alarmsFile.Close()
	}

	// IMPORT
	importService, err := service.NewImportService(s.influx, samplesFile, alarmsFile, samplesSize)
	if err != nil {
		jsonR.Encode(report.AddError(err))
		return
	}

	experiment, size, err := importService.ImportExperiment(experiment)
	if err != nil {
		jsonR.Encode(report.AddError(err))
		return
	}

	report.ExperimentID = experiment.ID
	report.Progress = int(int64(size*100) / samplesSize)

	go importService.ImportSamples(report, experiment, s.reports)
	go importService.ImportAlarms(report, experiment, s.reports)

	s.reports <- report
	jsonR.Encode(report)
}

func (s Server) eventsHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)

	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(200)
	i := 1

	for {
		report := <-s.reports

		b, err := json.Marshal(report)
		if err != nil {
			log.Fatal("Can't convert to JSON")
		}

		fmt.Println(b)
		fmt.Fprintf(w, "id: %d\n", i)
		fmt.Fprintf(w, "data: %s\n\n", b)

		i++
		flusher.Flush()
	}
}
