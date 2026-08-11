package main

import "fmt"

func main() {
	version, serviceName := "0.1.0", "Relay"

	fmt.Printf("service=%s version=%s\n", serviceName, version)

	event, err := NewEvent("")
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("event_type=%s\n", event.Type)
}
