package memory

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Bonay22/relay/internal/domain"
)

// TestEventMemory_SaveAndList проверяет сохранение, присвоение ID и чтение.
func TestEventMemory_SaveAndList(t *testing.T) {
	tests := []struct {
		name    string
		events  []domain.Event
		wantIDs []domain.EventID
	}{
		{
			name: "events",
			events: []domain.Event{
				{Type: domain.OrderCreated, Payload: map[string]any{"status": "ok"}, Status: domain.EventPending},
				{Type: domain.OrderCreated, Payload: map[string]any{"status": "ok"}, Status: domain.EventPending},
			},
			wantIDs: []domain.EventID{"event-1", "event-2"},
		},
		{
			name:    "empty",
			events:  []domain.Event{},
			wantIDs: []domain.EventID{},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			repo := NewEventRepository()
			savedEvents := make([]domain.Event, 0, len(testCase.events))

			for index, event := range testCase.events {
				saved, err := repo.Save(event)
				if err != nil {
					t.Fatalf("Save failed: %v", err)
				}

				if saved.ID != testCase.wantIDs[index] {
					t.Errorf("got ID = %s, want %s", saved.ID, testCase.wantIDs[index])
				}
				savedEvents = append(savedEvents, saved)
			}

			// Проверяем List
			result, err := repo.List()
			if err != nil {
				t.Fatalf("List failed: %v", err)
			}
			if result == nil {
				t.Fatal("List returned nil, want empty slice")
			}
			if len(result) != len(savedEvents) {
				t.Fatalf("List count = %d, want %d", len(result), len(savedEvents))
			}

			for index, got := range result {
				assertEventEquals(t, got, savedEvents[index])
			}
		})
	}
}

// TestEventMemory_SaveResultIsolation проверяет изоляцию результата Save от репозитория.
func TestEventMemory_SaveResultIsolation(t *testing.T) {
	repo := NewEventRepository()

	saved, err := repo.Save(domain.Event{
		Type:    domain.OrderCreated,
		Payload: newNestedPayload(),
	})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	customer := saved.Payload["customer"].(map[string]any)
	customer["name"] = "Anna"

	items := saved.Payload["items"].([]any)

	item := items[0].(map[string]any)
	item["code"] = "B-20"

	items[0] = "replacement"

	refreshed, err := repo.Get(saved.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	assertPayloadEqual(t, refreshed.Payload, newNestedPayload())
}

func TestEventMemory_GetResultIsolation(t *testing.T) {
	repo := NewEventRepository()
	event, err := domain.NewEvent(domain.OrderCreated, newNestedPayload())
	if err != nil {
		t.Fatal(err)
	}

	saved, err := repo.Save(event)
	if err != nil {
		t.Fatal(err)
	}

	getEvent, err := repo.Get(saved.ID)
	if err != nil {
		t.Fatal(err)
	}

	customer := getEvent.Payload["customer"].(map[string]any)
	customer["name"] = "Anna"

	items := getEvent.Payload["items"].([]any)

	item := items[0].(map[string]any)
	item["code"] = "B-20"

	items[0] = "replacement"

	fresh, err := repo.Get(saved.ID)
	if err != nil {
		t.Fatal(err)
	}

	assertPayloadEqual(t, fresh.Payload, newNestedPayload())
}

func TestEventMemory_ListIsolation(t *testing.T) {
	repo := NewEventRepository()

	saved, err := repo.Save(domain.Event{
		Type:    domain.OrderCreated,
		Status:  domain.EventPending,
		Payload: newNestedPayload(),
	})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	listed, err := repo.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("List count = %d, want 1", len(listed))
	}
	if listed[0].ID != saved.ID {
		t.Fatalf("listed ID = %s, want %s", listed[0].ID, saved.ID)
	}

	// Изменение элемента списка не должно менять сохранённое событие.
	customer := listed[0].Payload["customer"].(map[string]any)
	customer["name"] = "Anna"

	items := listed[0].Payload["items"].([]any)

	item := items[0].(map[string]any)
	item["code"] = "B-20"

	items[0] = "replacement"

	fresh, err := repo.Get(saved.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	assertPayloadEqual(t, fresh.Payload, newNestedPayload())
}

func TestEventMemory_SaveInputIsolation(t *testing.T) {
	repo := NewEventRepository()
	inputPayload := newNestedPayload()

	saved, err := repo.Save(domain.Event{
		Type:    domain.OrderCreated,
		Status:  domain.EventPending,
		Payload: inputPayload,
	})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	customer := inputPayload["customer"].(map[string]any)
	customer["name"] = "Anna"

	items := inputPayload["items"].([]any)
	items[0] = "replacement"

	fresh, err := repo.Get(saved.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	assertPayloadEqual(t, fresh.Payload, newNestedPayload())
}

// TestEventMemory_GetUnknownID проверяет ошибку чтения отсутствующего события.
func TestEventMemory_GetUnknownID(t *testing.T) {
	repo := NewEventRepository()

	event, err := repo.Get("unknown-event")
	checkingForValidError(t, err, domain.ErrEventNotFound)

	if !reflect.DeepEqual(event, domain.Event{}) {
		t.Errorf("expected empty domain.Event, got %+v", event)
	}
}

// --- Хелперы ---

func assertPayloadStatus(t *testing.T, payload map[string]any, wantStatus string) {
	t.Helper()
	val, ok := payload["status"]
	if !ok {
		t.Fatal("Payload is missing key \"status\"")
	}
	if val != wantStatus {
		t.Errorf("Payload[\"status\"] = %v, want %v", val, wantStatus)
	}
}

func assertEventEquals(t *testing.T, got, want domain.Event) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("event mismatch:\ngot:  %#v\nwant: %#v", got, want)
	}
	assertPayloadStatus(t, got.Payload, "ok")
}

func assertPayloadEqual(t *testing.T, got, want map[string]any) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Payload = %#v, want %#v", got, want)
	}
}

func checkingForValidError(t *testing.T, err, wantErr error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("unexpected error: %v, want %v", err, wantErr)
	}
}

func newNestedPayload() map[string]any {
	return map[string]any{
		"customer": map[string]any{
			"name": "Mira",
		},
		"items": []any{
			map[string]any{"code": "A-10"},
		},
	}
}
