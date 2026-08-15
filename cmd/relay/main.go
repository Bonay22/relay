package main

import "fmt"

func main() {
	version, serviceName := "0.1.0", "Relay"

	fmt.Printf("service=%s version=%s\n", serviceName, version)

	event, err := NewEvent(OrderCreated, map[string]any{"order_id": "A-10"})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("event_type=%s status=%s order_id=%v created_at_zero=%t\n", event.Type, event.Status, event.Payload["order_id"], event.CreatedAt.IsZero())
	err = event.MarkDelivered()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("event_status=%s\n", event.Status)

	invalidEvent, invalidErr := NewEvent("", nil)
	if invalidErr != nil {
		fmt.Printf("create_error=%v invalid_created_at_zero=%t\n", invalidErr, invalidEvent.CreatedAt.IsZero())
	}
}
