package main

import (
	"log"

	"github.com/leaklessgfy/safran-server/server"
)

func main() {
	server := server.NewServer()

	log.Println("Start Server on :8888")
	err := server.Start(":8888")
	if err != nil {
		log.Fatal(err)
	}
}
