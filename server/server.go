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
	imports map[string]chan entity.Report
}

// NewServer create a server instance
func NewServer() (*Server, error) {
	influx, err := service.NewInfluxService()
	if err != nil {
		return nil, err
	}

	imports := make(map[string]chan entity.Report)
	imports["TEST"] = make(chan entity.Report, 2)

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
	http.HandleFunc("/size", s.sizeHandler)
	http.HandleFunc("/install", s.installHandler)
	http.HandleFunc("/drop", s.dropHandler)
	log.Println("Server Start on :8888")

	return http.ListenAndServe(":8888", nil)
}

func (s Server) simpleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	select {
	case s.imports["TEST"] <- *entity.NewReport("TEST"):
		w.WriteHeader(200)
	default:
		http.Error(w, "no consumer", http.StatusNotFound)
	}
}

func (s Server) uploadHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	r.ParseMultipartForm(32 << 20)

	channelUUID, err := uuid.NewV4()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	channelID := channelUUID.String()
	jsonR := json.NewEncoder(w)
	report := entity.NewReport(channelID)

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

	channel := make(chan entity.Report, 50)
	s.imports[channelID] = channel

	go importService.ImportSamples(*report.Copy(entity.ReportTypeSamples), *experiment, channel)
	go importService.ImportAlarms(*report.Copy(entity.ReportTypeAlarms), *experiment, channel)
	go importService.Save(*report.Copy(entity.ReportTypeClient), channel)

	jsonR.Encode(report)
}

func (s Server) eventsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	channelID := r.URL.Query().Get("channel")
	channel, ok := s.imports[channelID]
	if !ok {
		http.Error(w, "Undefined channel "+channelID, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		select {
		case <-r.Context().Done():
			return
		case report := <-channel:
			fmt.Fprintf(w, "id: %d\n", report.ID)
			fmt.Fprintf(w, "event: %s\n", report.Type)
			fmt.Fprintf(w, "data: %s\n\n", report.ToJSON())
			flusher.Flush()

			if report.HasComplete() && report.Type == entity.ReportTypeClient {
				close(channel)
				delete(s.imports, channelID)
				return
			}
		}
	}
}

func (s Server) sizeHandler(w http.ResponseWriter, r *http.Request) {
	size, err := s.influx.Size()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Write([]byte(size))
	}
}

func (s Server) installHandler(w http.ResponseWriter, r *http.Request) {
	err := s.influx.Install()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Write([]byte("done\n"))
	}
}

func (s Server) dropHandler(w http.ResponseWriter, r *http.Request) {
	err := s.influx.Drop()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Write([]byte("done\n"))
	}
}
