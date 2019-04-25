package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/leaklessgfy/safran-server/entity"

	"github.com/leaklessgfy/safran-server/service"
)

// Server is an abstraction layer for http server
type Server struct {
	message chan []byte
}

// Response is the struct representing a response
type Response struct {
	Err bool   `json:"error"`
	Msg string `json:"msg"`
}

// NewServer create a server instance
func NewServer() *Server {
	return &Server{message: make(chan []byte)}
}

// Start will start the http server and setup routes
func (s Server) Start() error {
	http.HandleFunc("/upload", s.uploadHandler)
	http.HandleFunc("/events", s.eventsHandler)
	log.Println("Server Start on :8888")
	return http.ListenAndServe(":8888", nil)
}

func (s Server) uploadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)

	experimentValue := r.FormValue("experiment")
	if experimentValue == "" {
		json.NewEncoder(w).Encode(Response{Err: true, Msg: "experiment is required"})
		return
	}

	var experiment entity.Experiment
	err := json.Unmarshal([]byte(experimentValue), &experiment)
	if err != nil {
		json.NewEncoder(w).Encode(Response{Err: true, Msg: err.Error()})
		return
	}

	samplesFile, _, err := r.FormFile("samples")
	if err != nil {
		json.NewEncoder(w).Encode(Response{Err: true, Msg: "samplesFile is required"})
		return
	}
	alarmsFile, _, _ := r.FormFile("alarms")
	defer samplesFile.Close()
	if alarmsFile != nil {
		defer alarmsFile.Close()
	}

	go s.handleImport(experiment, samplesFile, alarmsFile)

	json.NewEncoder(w).Encode(Response{Err: false, Msg: "success"})
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
	for {
		fmt.Fprintf(w, "data: %s\n\n", <-s.message)
		flusher.Flush()
	}
}

func (s Server) handleImport(experiment entity.Experiment, samplesFile, alarmsFile io.Reader) {
	importService, err := service.NewImportService()
	if err != nil {
		// Handle error in specific events ?
		log.Println("[New Import Service] - ", err)
		return
	}
	experiment, err = importService.ImportExperiment(experiment, samplesFile)
	if err != nil {
		log.Println("[Import Experiment] - ", err)
		// Handle error in specific events ?
		return
	}
	err = importService.ImportAlarms(experiment, alarmsFile)
	if err != nil {
		log.Println("[Import Alarms] - ", err)
		// Handle error in specific events ?
	}
}
