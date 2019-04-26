package main

import (
	"log"

	"github.com/leaklessgfy/safran-server/server"
)

func main() {
	server, err := server.NewServer()
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(server.Start())
}
