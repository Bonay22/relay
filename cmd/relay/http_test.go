package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Bonay22/relay/internal/application"
	"github.com/Bonay22/relay/internal/domain"
	"github.com/Bonay22/relay/internal/memory"
)

func newTestRouter() (http.Handler, *application.EventService) {
	repository := memory.NewEventRepository()
	service := application.NewEventService(repository)

	return newRouter(service), service
}

// TestDecodeCreateEventRequest проверяет строгий разбор тела запроса без HTTP-слоя.
func TestDecodeCreateEventRequest(t *testing.T) {
	tests := []struct {
		name      string
		inputJSON string
		wantErr   bool
		wantErrIs error
	}{
		{
			name:      "valid",
			inputJSON: `{"type": "order.created", "payload": {"order_id": "A-10", "customer": "Мира", "amount": 9007199254740993}}`,
			wantErr:   false,
			wantErrIs: nil,
		},
		{
			name:      "malformed_json",
			inputJSON: `{"type": "order.created", "payload": {`,
			wantErr:   true,
			wantErrIs: nil,
		},
		{
			name:      "unknown_field",
			inputJSON: `{"type": "order.created", "payload": {}, "some_unknown_field": 123}`,
			wantErr:   true,
			wantErrIs: nil,
		},
		// Второй объект не даёт проверочному Decode получить ожидаемый io.EOF.
		{
			name: "multiple_json_objects",
			inputJSON: `{"type":"order.created","payload":{}}
{"type":"order.paid","payload":{}}`,
			wantErr:   true,
			wantErrIs: ErrSingleJSONObjectRequired,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			body := strings.NewReader(testCase.inputJSON)
			gotRequest, err := decodeCreateEventRequest(body)

			if testCase.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if testCase.wantErrIs != nil {
					// errors.Is проверяет принадлежность к sentinel error, а не текст ошибки.
					if !errors.Is(err, testCase.wantErrIs) {
						t.Fatalf("unexpected error: %v", err)
					}
				}

				return
			}

			requireNoError(t, err)

			orderID, ok := gotRequest.Payload["order_id"]

			if gotRequest.Type != "order.created" {
				t.Errorf("expected Type to be 'order.created', got '%s'", gotRequest.Type)
			}

			if !ok {
				t.Fatal("expected 'order_id' to exist in Payload")
			}

			if orderID != "A-10" {
				t.Errorf("expected Payload['order_id'] to be 'A-10', got '%v'", orderID)
			}

			// UseNumber сохраняет JSON-число без преобразования во float64.
			value := gotRequest.Payload["amount"]
			// Type assertion проверяет динамический тип значения внутри any.
			amount, ok := value.(json.Number)

			if !ok {
				t.Fatalf("amount has type %T, want json.Number", value)
			}

			if amount.String() != "9007199254740993" {
				t.Errorf("amount.String() == %s, want \"9007199254740993\"", amount.String())
			}

			value, ok = gotRequest.Payload["customer"]
			if !ok {
				t.Fatal("expected 'customer' to exist in Payload")
			}

			// Значение из map[string]any требует type assertion к строке.
			customer, ok := value.(string)
			if !ok {
				t.Fatalf("customer has type %T, want string", value)
			}

			if customer != "Мира" {
				t.Errorf("customer == %s, want \"Мира\"", customer)
			}
		})
	}
}

// TestCreateEventResponse проверяет публичные ответы для некорректных запросов.
func TestCreateEventResponse(t *testing.T) {
	tests := []struct {
		name        string
		inputJSON   string
		wantStatus  int
		wantMessage string
	}{
		{
			name:        "malformed_json",
			inputJSON:   `{"type": "order.created", "payload": {`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "invalid request body",
		},
		{
			name:        "empty_type",
			inputJSON:   `{"type": ""}`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "event type is empty",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			router, _ := newTestRouter()

			body := strings.NewReader(testCase.inputJSON)
			request := httptest.NewRequest(http.MethodPost, "/v1/events", body)
			// Recorder реализует ResponseWriter и сохраняет ответ без открытия TCP-порта.
			recorder := httptest.NewRecorder()

			// ServeHTTP проверяет тот же router и выбор маршрута, что использует сервер.
			router.ServeHTTP(recorder, request)

			assertJSONResponse(t, recorder, testCase.wantStatus)

			response := decodeJSON[createErrorResponse](t, recorder.Body)

			if response.Error != testCase.wantMessage {
				t.Errorf("error message = %s, want %s", response.Error, testCase.wantMessage)
			}
		})
	}
}

// TestHealth защищает контракт GET /health.
func TestHealth(t *testing.T) {
	router, _ := newTestRouter()

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertJSONResponse(t, recorder, http.StatusOK)

	// Encoder добавляет к JSON завершающий перевод строки.
	body := recorder.Body.String()
	if body != "{\"status\":\"ok\"}\n" {
		t.Errorf(
			"body = %q, want %q",
			body,
			"{\"status\":\"ok\"}\n",
		)
	}
}

// TestCreateEvent проверяет успешный контракт POST /v1/events целиком.
func TestCreateEvent(t *testing.T) {
	router, service := newTestRouter()

	body := strings.NewReader(`{"type":"order.created","payload":{"order_id":"A-10"}}`)
	request := httptest.NewRequest(http.MethodPost, "/v1/events", body)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)
	assertJSONResponse(t, recorder, http.StatusCreated)

	response := decodeJSON[createEventResponse](t, recorder.Body)
	assertResponseCheck(t, response, "event-1", "A-10")

	getEvent, err := service.Get(response.ID)

	requireNoError(t, err)
	assertEventCheck(t, getEvent, "event-1", "A-10")

	body2 := strings.NewReader(`{"type":"order.created","payload":{"order_id":"B-20"}}`)
	recorder2 := httptest.NewRecorder()
	request2 := httptest.NewRequest(http.MethodPost, "/v1/events", body2)

	router.ServeHTTP(recorder2, request2)
	assertJSONResponse(t, recorder2, http.StatusCreated)

	response2 := decodeJSON[createEventResponse](t, recorder2.Body)
	assertResponseCheck(t, response2, "event-2", "B-20")

	getEvent2, err := service.Get(response2.ID)
	requireNoError(t, err)
	assertEventCheck(t, getEvent2, "event-2", "B-20")
}

// requireNoError завершает текущий тест, если вызов неожиданно вернул ошибку.
func requireNoError(t *testing.T, err error) {
	// Helper сообщает testing о вспомогательной функции, чтобы ошибка указывала
	// на строку вызова requireNoError.
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus int) {
	t.Helper()

	if recorder.Code != wantStatus {
		t.Fatalf(
			"status code = %d, want %d",
			recorder.Code,
			wantStatus,
		)
	}

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf(
			"Content-Type = %q, want %q",
			contentType,
			"application/json",
		)
	}
}

func assertEventCheck(t *testing.T, event domain.Event, wantID domain.EventID, wantOrderID string) {
	t.Helper()

	if event.Type != domain.OrderCreated {
		t.Errorf("Type = %s, want %s", event.Type, domain.OrderCreated)
	}

	if event.Status != domain.EventPending {
		t.Errorf("Status = %s, want %s", event.Status, domain.EventPending)
	}

	if event.Payload["order_id"] != wantOrderID {
		t.Errorf("Payload = %s, want %s", event.Payload["order_id"], wantOrderID)
	}

	if event.CreatedAt.IsZero() {
		t.Error("event.CreatedAt is zero time")
	}

	if event.ID != wantID {
		t.Errorf("ID = %s, want %s", event.ID, wantID)
	}
}

func assertResponseCheck(t *testing.T, response createEventResponse, wantID domain.EventID, wantOrderID string) {
	t.Helper()

	if response.Type != domain.OrderCreated {
		t.Errorf("Type = %s, want %s", response.Type, domain.OrderCreated)
	}

	if response.Status != domain.EventPending {
		t.Errorf("Status = %s, want %s", response.Status, domain.EventPending)
	}

	if response.Payload["order_id"] != wantOrderID {
		t.Errorf("Payload = %s, want %s", response.Payload["order_id"], wantOrderID)
	}

	if response.CreatedAt.IsZero() {
		t.Error("response.CreatedAt is zero time")
	}

	if response.ID != wantID {
		t.Errorf("ID = %s, want %s", response.ID, wantID)
	}
}

func decodeJSON[T any](t *testing.T, body io.Reader) T {
	t.Helper()

	var result T
	err := json.NewDecoder(body).Decode(&result)
	requireNoError(t, err)

	return result
}
