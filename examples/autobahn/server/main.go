package main

import (
	"log"
	"net/http"

	"gopkg.in/beatgammit/turnpike.v2"
)

func main() {
	turnpike.Debug()
	s := turnpike.NewBasicWebsocketServer("turnpike.examples")
	server := &http.Server{
		Handler: s,
		Addr:    ":8000",
	}
	log.Info("turnpike server starting on port 8000")
	log.Fatal(server.ListenAndServe())
}
