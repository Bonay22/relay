package main

import (
	"errors"
	"testing"
)

func TestNewEvent(t *testing.T) {
	tests := []struct {
		name           string
		eventType      EventType
		payload        map[string]any
		wantPayloadKey string
		wantPayload    string
		wantNilPayload bool
	}{
		{
			name:           "order_created",
			eventType:      OrderCreated,
			payload:        map[string]any{"order_id": "A-10"},
			wantPayloadKey: "order_id",
			wantPayload:    "A-10",
			wantNilPayload: false,
		},
		{
			name:           "order_paid",
			eventType:      OrderPaid,
			payload:        map[string]any{"payment_id": "P-20"},
			wantPayloadKey: "payment_id",
			wantPayload:    "P-20",
			wantNilPayload: false,
		},
		{
			name:           "nil_payload",
			eventType:      OrderCreated,
			payload:        nil,
			wantPayloadKey: "",
			wantPayload:    "",
			wantNilPayload: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			event, err := NewEvent(testCase.eventType, testCase.payload)
			requireNoError(t, err)

			if event.Type != testCase.eventType {
				t.Errorf("event.Type = %v, want %v", event.Type, testCase.eventType)
			}

			if event.Status != EventPending {
				t.Errorf("event.Status = %v, want %v", event.Status, EventPending)
			}

			if event.CreatedAt.IsZero() {
				t.Error("event.CreatedAt.IsZero() = true, want false")
			}

			if testCase.wantNilPayload {
				if event.Payload != nil {
					t.Errorf("event.Payload = %v, want nil", event.Payload)
				}
				return
			}

			val, exists := event.Payload[testCase.wantPayloadKey]
			if !exists {
				t.Fatalf("event.Payload is missing key %q", testCase.wantPayloadKey)
			}

			if val != testCase.wantPayload {
				t.Errorf("event.Payload[%q] = %v, want %v", testCase.wantPayloadKey, val, testCase.wantPayload)
			}
		})
	}
}

func TestNewEventRejectsEmptyType(t *testing.T) {
	event, err := NewEvent("", nil)

	if !errors.Is(err, ErrEventTypeEmpty) {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.Type != "" {
		t.Errorf("event.Type = %v, want \"\"", event.Type)
	}

	if event.Status != "" {
		t.Errorf("event.Status = %v, want \"\"", event.Status)
	}

	if event.Payload != nil {
		t.Errorf("event.Payload = %v, want nil", event.Payload)
	}

	if !event.CreatedAt.IsZero() {
		t.Errorf("event.CreatedAt = %v, want zero time", event.CreatedAt)
	}
}

func TestEventMarkDelivered(t *testing.T) {
	event, err := NewEvent(OrderCreated, map[string]any{"order_id": "A-10"})
	requireNoError(t, err)

	if event.Status != EventPending {
		t.Errorf("event.Status = %v, want %v", event.Status, EventPending)
	}

	err = event.MarkDelivered()
	requireNoError(t, err)

	if event.Status != EventDelivered {
		t.Errorf("event.Status = %v, want %v", event.Status, EventDelivered)
	}
}

func TestEventMarkDeliveredNilReceiver(t *testing.T) {
	var event *Event
	err := event.MarkDelivered()

	if !errors.Is(err, ErrEventNil) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
