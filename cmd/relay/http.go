package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

var ErrSingleJSONObjectRequired = errors.New("body must contain only one JSON object")

// createEventRequest описывает допустимые поля JSON-запроса на создание события.
type createEventRequest struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

// createEventResponse описывает JSON-ответ с созданным событием.
type createEventResponse struct {
	Type      EventType      `json:"type"`
	Status    EventStatus    `json:"status"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}

// createErrorResponse задаёт единый формат ошибочного JSON-ответа.
type createErrorResponse struct {
	Error string `json:"error"`
}

// writeErrorResponse записывает ошибку в едином JSON-формате.
func writeErrorResponse(w http.ResponseWriter, status int, message string) {
	// Заголовки нужно задать до WriteHeader: после него HTTP-статус уже зафиксирован.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorResponse := createErrorResponse{Error: message}

	err := json.NewEncoder(w).Encode(errorResponse)
	if err != nil {
		return
	}
}

// decodeCreateEventRequest декодирует ровно один JSON-объект и отклоняет неизвестные поля.
func decodeCreateEventRequest(body io.Reader) (createEventRequest, error) {
	var request createEventRequest

	decoder := json.NewDecoder(body)
	// DisallowUnknownFields защищает внешний контракт от незаявленных полей DTO.
	decoder.DisallowUnknownFields()
	// UseNumber сохраняет числа внутри any как json.Number вместо float64.
	decoder.UseNumber()

	err := decoder.Decode(&request)
	if err != nil {
		return createEventRequest{}, err
	}

	// Используем тот же decoder: часть входа уже может находиться в его внутреннем буфере.
	// Второй Decode должен получить io.EOF; nil означает, что найдено второе JSON-значение.
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

// newRouter собирает все HTTP-маршруты сервиса.
func newRouter() http.Handler {
	router := http.NewServeMux()

	router.HandleFunc("GET /health", healthHandler)
	router.HandleFunc("POST /v1/events", createEventHandler)

	return router
}

// healthHandler возвращает состояние сервиса для GET /health.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	if err != nil {
		return
	}
}

// createEventHandler создаёт событие по запросу POST /v1/events.
func createEventHandler(w http.ResponseWriter, r *http.Request) {
	request, err := decodeCreateEventRequest(r.Body)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Преобразование в EventType меняет тип строки, но доменную валидацию выполняет NewEvent.
	event, err := NewEvent(EventType(request.Type), request.Payload)
	if err != nil {
		// errors.Is распознаёт sentinel error даже после возможного оборачивания.
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
