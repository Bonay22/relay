package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	version, serviceName := "0.1.0", "Relay"
	fmt.Printf("service=%s version=%s\n", serviceName, version)

	const address = "127.0.0.1:8080"
	router := newRouter()

	log.Printf("server listening on %s", address)

	err := http.ListenAndServe(address, router)
	if err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
