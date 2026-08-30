package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
			router := newRouter()

			body := strings.NewReader(testCase.inputJSON)
			request := httptest.NewRequest(http.MethodPost, "/v1/events", body)
			// Recorder реализует ResponseWriter и сохраняет ответ без открытия TCP-порта.
			recorder := httptest.NewRecorder()

			// ServeHTTP проверяет тот же router и выбор маршрута, что использует сервер.
			router.ServeHTTP(recorder, request)

			if recorder.Code != testCase.wantStatus {
				t.Errorf("status code = %d, want %d", recorder.Code, testCase.wantStatus)
			}

			contentType := recorder.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf(
					"Content-Type = %q, want %q",
					contentType,
					"application/json",
				)
			}

			var response createErrorResponse

			err := json.NewDecoder(recorder.Body).Decode(&response)
			requireNoError(t, err)

			if response.Error != testCase.wantMessage {
				t.Errorf("error message = %s, want %s", response.Error, testCase.wantMessage)
			}
		})
	}
}

// TestHealth защищает контракт GET /health.
func TestHealth(t *testing.T) {
	router := newRouter()

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Errorf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusOK,
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
	router := newRouter()

	body := strings.NewReader(`{"type":"order.created","payload":{"order_id":"A-10"}}`)
	request := httptest.NewRequest(http.MethodPost, "/v1/events", body)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusCreated,
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

	var response createEventResponse
	err := json.NewDecoder(recorder.Body).Decode(&response)

	requireNoError(t, err)

	if response.Type != OrderCreated {
		t.Errorf("Type = %s, want %s", response.Type, OrderCreated)
	}

	if response.Status != EventPending {
		t.Errorf("Status = %s, want %s", response.Status, EventPending)
	}

	if response.Payload["order_id"] != "A-10" {
		t.Errorf("Payload = %s, want %s", response.Payload["order_id"], "A-10")
	}

	if response.CreatedAt.IsZero() {
		t.Error("event.CreatedAt is zero time")
	}
}
