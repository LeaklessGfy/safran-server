package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/leaklessgfy/safran-server/entity"
	uuid "github.com/satori/go.uuid"

	"github.com/leaklessgfy/safran-server/service"
)

// Server is an abstraction layer for http server
type Server struct {
	influx  *service.InfluxService
	imports map[string]Channel
}

type Channel struct {
	channel chan entity.Report
}

// NewServer create a server instance
func NewServer() (*Server, error) {
	influx, err := service.NewInfluxService()
	if err != nil {
		return nil, err
	}

	imports := make(map[string]Channel)
	imports["TEST"] = Channel{channel: make(chan entity.Report, 2)}

	return &Server{
		influx:  influx,
		imports: imports,
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
	case s.imports["TEST"].channel <- *entity.NewReport("TEST"):
		w.WriteHeader(200)
	default:
		http.Error(w, "no consumer", http.StatusNotFound)
	}
}

func (s Server) uploadHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	r.ParseMultipartForm(32 << 20)

	channel, err := uuid.NewV4()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonR := json.NewEncoder(w)
	report := entity.NewReport(channel.String())

	// EXPERIMENT
	experiment, err := service.ExtractExperiment(r)
	if err != nil {
		jsonR.Encode(report.AddError(entity.ReportStepExtractExperiment, err))
		return
	}
	report.AddSuccess(entity.ReportStepExtractExperiment)

	// FILES
	samplesFile, samplesSize, err := service.ExtractSamples(r)
	if err != nil {
		jsonR.Encode(report.AddError(entity.ReportStepExtractSamples, err))
		return
	}
	report.SamplesSize = samplesSize
	report.AddSuccess(entity.ReportStepExtractSamples)
	defer samplesFile.Close()

	alarmsFile, alarmsSize, err := service.ExtractAlarms(r)
	if err != nil {
		jsonR.Encode(report.AddError(entity.ReportStepExtractAlarms, err))
		return
	}
	if alarmsFile != nil {
		report.HasAlarms = true
		report.AlarmsSize = alarmsSize
		report.AddSuccess(entity.ReportStepExtractAlarms)
		defer alarmsFile.Close()
	}

	// IMPORT
	importService, err := service.NewImportService(s.influx, samplesFile, alarmsFile)
	if err != nil {
		jsonR.Encode(report.AddError(entity.ReportStepInitImport, err))
		return
	}
	report.AddSuccess(entity.ReportStepInitImport)

	err = importService.ImportExperiment(report, experiment)
	if err != nil {
		jsonR.Encode(report)
		return
	}

	c := make(chan entity.Report, 50)
	s.imports[channel.String()] = Channel{channel: c}

	go importService.ImportSamples(*report.Copy(entity.ReportTypeSamples), *experiment, c)
	go importService.ImportAlarms(*report.Copy(entity.ReportTypeAlarms), *experiment, c)
	go importService.Save(*report.Copy(entity.ReportTypeClient), c)

	jsonR.Encode(report)
}

func (s Server) eventsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	channel := r.URL.Query().Get("channel")
	if _, ok = s.imports[channel]; !ok {
		http.Error(w, "Undefined channel "+channel, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	c := s.imports[channel]

	for {
		select {
		case <-r.Context().Done():
			return
		case report := <-c.channel:
			json := report.ToJSON()
			fmt.Fprintf(w, "id: %d\n", report.ID)
			fmt.Fprintf(w, "event: %s\n", report.Type)
			fmt.Fprintf(w, "data: %s\n\n\n", json)
			fmt.Println(report)
			flusher.Flush()

			if report.HasComplete() {
				if report.Type == entity.ReportTypeClient {
					close(c.channel)
					delete(s.imports, channel)
					return
				}
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
