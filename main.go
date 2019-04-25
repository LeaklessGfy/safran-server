package main

import (
	"log"

	"github.com/leaklessgfy/safran-server/server"
)

func main() {
	server := server.NewServer()
	log.Fatal(server.Start())
}
