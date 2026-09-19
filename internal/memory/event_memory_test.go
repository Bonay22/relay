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

// TestEventMemory_PayloadIsolation проверяет, что мутация Payload во внешней переменной
// или в полученном объекте не меняет данные внутри репозитория (копирование по значению/глубокая копия).
func TestEventMemory_PayloadIsolation(t *testing.T) {
	repo := NewEventRepository()

	saved, err := repo.Save(domain.Event{
		Type:    domain.OrderCreated,
		Payload: map[string]any{"status": "ok"},
	})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 1. Изменяем сохраненный объект локально
	saved.Payload["status"] = "mutated"

	// 2. Получаем объект из репозитория и проверяем, что старый статус не изменился
	fresh, err := repo.Get(saved.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	assertPayloadStatus(t, fresh.Payload, "ok")

	// 3. Изменяем полученный объект и проверяем репозиторий еще раз
	fresh.Payload["status"] = "mutated_again"

	freshAfterMutation, err := repo.Get(saved.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	assertPayloadStatus(t, freshAfterMutation.Payload, "ok")
}

func TestEventMemory_GetIsolation(t *testing.T) {
	repo := NewEventRepository()
	event, err := domain.NewEvent(domain.OrderCreated, map[string]any{"status": "ok"})
	if err != nil {
		t.Fatal(err)
	}

	saved, err := repo.Save(event)
	if err != nil {
		t.Fatal(err)
	}

	event.Payload["status"] = "mutated"

	fresh, err := repo.Get(saved.ID)
	if err != nil {
		t.Fatal(err)
	}

	assertPayloadStatus(t, fresh.Payload, "ok")
}

func TestEventMemory_ListIsolation(t *testing.T) {
	repo := NewEventRepository()

	saved, err := repo.Save(domain.Event{
		Type:    domain.OrderCreated,
		Status:  domain.EventPending,
		Payload: map[string]any{"status": "ok"},
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
	listed[0].Payload["status"] = "changed"

	fresh, err := repo.Get(saved.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	assertPayloadStatus(t, fresh.Payload, "ok")
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

func checkingForValidError(t *testing.T, err, wantErr error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("unexpected error: %v, want %v", err, wantErr)
	}
}
