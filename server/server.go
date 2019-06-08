package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/leaklessgfy/safran-server/observer"

	"github.com/leaklessgfy/safran-server/entity"
	"github.com/leaklessgfy/safran-server/facade"
	uuid "github.com/satori/go.uuid"

	"github.com/leaklessgfy/safran-server/service"
)

// Server is an abstraction layer for http server
type Server struct {
	imports map[string]chan entity.Report
}

// NewServer create a server instance
func NewServer() *Server {
	imports := make(map[string]chan entity.Report)
	imports["TEST"] = make(chan entity.Report, 2)

	return &Server{
		imports: imports,
	}
}

// Start will start the http server and setup routes
func (s Server) Start(port string) error {
	http.HandleFunc("/simple", s.simpleHandler)
	http.HandleFunc("/upload", s.uploadHandler)
	http.HandleFunc("/events", s.eventsHandler)

	return http.ListenAndServe(port, nil)
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
	w.Header().Set("Content-Type", "application/json")
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
		jsonR.Encode(report.AddError(entity.StepExtractExperiment, err))
		return
	}
	report.AddSuccess(entity.StepExtractExperiment)

	// SAVER
	saver, err := service.ExtractSaver(r)
	if err != nil {
		jsonR.Encode(report.AddError(entity.StepExtractSaver, err))
		return
	}
	report.AddSuccess(entity.StepExtractSaver)

	// FILES
	samplesFile, samplesSize, err := service.ExtractSamples(r)
	if err != nil {
		jsonR.Encode(report.AddError(entity.StepExtractSamples, err))
		return
	}
	report.SamplesSize = samplesSize
	report.AddSuccess(entity.StepExtractSamples)
	defer samplesFile.Close()

	alarmsFile, alarmsSize, err := service.ExtractAlarms(r)
	if err != nil {
		jsonR.Encode(report.AddError(entity.StepExtractAlarms, err))
		return
	}
	if alarmsFile != nil {
		report.HasAlarms = true
		report.AlarmsSize = alarmsSize
		report.AddSuccess(entity.StepExtractAlarms)
		defer alarmsFile.Close()
	}

	// IMPORT
	observer := observer.LoggerObserver{}
	facade := facade.NewParserFacade(saver, observer, samplesFile, alarmsFile)

	err = facade.Parse(experiment)
	if err != nil {
		jsonR.Encode(report.AddError(entity.StepInitImport, err))
		return
	}
	report.AddSuccess(entity.StepInitImport)

	channel := make(chan entity.Report, 50)
	s.imports[channelID] = channel

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

			if report.HasComplete() && report.Type == entity.TypeClient {
				close(channel)
				delete(s.imports, channelID)
				return
			}
		}
	}
}
