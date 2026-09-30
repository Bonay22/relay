package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Bonay22/relay/internal/application"
	"github.com/Bonay22/relay/internal/memory"
)

const (
	version     = "0.2.0"
	serviceName = "Relay"
	address     = "127.0.0.1:8080"
)

func main() {
	fmt.Printf("service=%s version=%s\n", serviceName, version)

	repository := memory.NewEventRepository()
	service := application.NewEventService(repository)
	router := newRouter(service)

	log.Printf("server listening on %s", address)

	err := http.ListenAndServe(address, router)
	if err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
