package main

import (
	"fmt"
	"log"
	"net/http"
)

const (
	version     = "0.2.0"
	serviceName = "Relay"
	address     = "127.0.0.1:8080"
)

func main() {
	fmt.Printf("service=%s version=%s\n", serviceName, version)

	router := newRouter()
	log.Printf("server listening on %s", address)

	err := http.ListenAndServe(address, router)
	if err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
