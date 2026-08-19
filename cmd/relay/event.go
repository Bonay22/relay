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

var (
	ErrEventTypeEmpty = errors.New("event type is empty")
	ErrEventNil       = errors.New("event is nil")
)

type Event struct {
	Type      EventType
	Status    EventStatus
	Payload   map[string]any
	CreatedAt time.Time
}

func NewEvent(eventType EventType, payload map[string]any) (Event, error) {
	if eventType == "" {
		return Event{}, ErrEventTypeEmpty
	}

	return Event{Type: eventType, Status: EventPending, Payload: payload, CreatedAt: time.Now()}, nil
}

func (event *Event) MarkDelivered() error {
	if event == nil {
		return ErrEventNil
	}

	event.Status = EventDelivered
	return nil
}
