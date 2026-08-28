package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

var ErrSingleJSONObjectRequired = errors.New("body must contain only one JSON object")

type createEventRequest struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

type createEventResponse struct {
	Type      EventType      `json:"type"`
	Status    EventStatus    `json:"status"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}

type createErrorResponse struct {
	Error string `json:"error"`
}

func writeErrorResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorResponse := createErrorResponse{Error: message}

	err := json.NewEncoder(w).Encode(errorResponse)
	if err != nil {
		return
	}
}

func decodeCreateEventRequest(body io.Reader) (createEventRequest, error) {
	var request createEventRequest
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	decoder.UseNumber()

	err := decoder.Decode(&request)
	if err != nil {
		return createEventRequest{}, err
	}

	var second any
	err = decoder.Decode(&second)

	if err == nil {
		return createEventRequest{}, ErrSingleJSONObjectRequired
	}
	if err != io.EOF {
		return createEventRequest{}, err
	}

	return request, nil
}

func newRouter() http.Handler {
	router := http.NewServeMux()

	router.HandleFunc("GET /health", healthHandler)
	router.HandleFunc("POST /v1/events", createEventHandler)

	return router
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Кодируем и отправляем JSON
	err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	if err != nil {
		return
	}
}

func createEventHandler(w http.ResponseWriter, r *http.Request) {
	request, err := decodeCreateEventRequest(r.Body)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	event, err := NewEvent(EventType(request.Type), request.Payload)
	if err != nil {
		switch {
		case errors.Is(err, ErrEventTypeEmpty):
			writeErrorResponse(w, http.StatusBadRequest, "event type is empty")
		default:
			writeErrorResponse(w, http.StatusInternalServerError, "internal server error")
		}

		return
	}

	eventResponse := createEventResponse{
		Type:      event.Type,
		Status:    event.Status,
		Payload:   event.Payload,
		CreatedAt: event.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(eventResponse)
	if err != nil {
		return
	}
}
