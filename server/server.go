package server

import (
	"encoding/json"
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
		reports: make(chan entity.Report, 10),
		influx:  influx,
	}, nil
}

// Start will start the http server and setup routes
func (s Server) Start() error {
	http.HandleFunc("/simple", s.simpleHandler)
	http.HandleFunc("/upload", s.uploadHandler)
	http.HandleFunc("/events", s.eventsHandler)
	http.HandleFunc("/install", s.installHandler)
	http.HandleFunc("/drop", s.dropHandler)
	log.Println("Server Start on :8888")

	return http.ListenAndServe(":8888", nil)
}

func (s Server) simpleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	select {
	case s.reports <- entity.NewReport():
		w.WriteHeader(200)
	default:
		http.Error(w, "no consumer", http.StatusNotFound)
	}
}

func (s Server) uploadHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	r.ParseMultipartForm(32 << 20)

	jsonR := json.NewEncoder(w)
	report := entity.NewReport()

	// EXPERIMENT
	experiment, err := service.ExtractExperiment(r)
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

	alarmsFile, alarmsSize, err := service.ExtractAlarms(r)
	if err != nil {
		jsonR.Encode(report.AddError(err))
		return
	}
	if alarmsFile != nil {
		report.HasAlarms = true
		defer alarmsFile.Close()
	}

	// IMPORT
	importService, err := service.NewImportService(s.influx, samplesFile, alarmsFile, samplesSize, alarmsSize, s.reports)
	if err != nil {
		jsonR.Encode(report.AddError(err))
		return
	}

	err = importService.ImportExperiment(&report, experiment)
	if err != nil {
		jsonR.Encode(report.AddError(err))
		return
	}

	s.reports <- report

	go importService.ImportSamples(report, *experiment)
	go importService.ImportAlarms(report, *experiment)

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
	i := 1

	for {
		select {
		case <-r.Context().Done():
			return
		case report := <-s.reports:
			b, err := json.Marshal(report)

			if err != nil {
				log.Fatal("Can't convert to JSON")
			}

			fmt.Fprintf(w, "id: %d\n", i)
			fmt.Fprintf(w, "data: %s\n\n", b)

			i++
			flusher.Flush()

			if report.Status == entity.StatusFail || report.Status == entity.StatusSuccess {
				return
			}
		}
	}
}

func (s Server) installHandler(w http.ResponseWriter, r *http.Request) {
	err := s.influx.Install()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("done\n"))
}

func (s Server) dropHandler(w http.ResponseWriter, r *http.Request) {
	err := s.influx.Drop()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("done\n"))
}
