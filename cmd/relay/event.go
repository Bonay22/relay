package main

import (
	"errors"
	"time"
)

type (
	EventType   string
	EventStatus string
)

const (
	OrderCreated EventType = "order.created"
	OrderPaid    EventType = "order.paid"

	EventPending   EventStatus = "pending"
	EventDelivered EventStatus = "delivered"
)

type Event struct {
	Type      EventType
	Status    EventStatus
	Payload   map[string]any
	CreatedAt time.Time
}

func NewEvent(eventType EventType, payload map[string]any) (Event, error) {
	if eventType == "" {
		return Event{}, errors.New("event type is empty")
	}

	return Event{Type: eventType, Status: EventPending, Payload: payload, CreatedAt: time.Now()}, nil
}

func (event *Event) MarkDelivered() error {
	if event == nil {
		return errors.New("event is nil")
	}

	event.Status = EventDelivered
	return nil
}
