package application

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Bonay22/relay/internal/domain"
)

var (
	ErrSaveFailed = errors.New("save failed")
	ErrRepository = errors.New("repository error")
)

// fakeEventRepository записывает аргументы методов и возвращает заранее заданные результаты.
type fakeEventRepository struct {
	saveCalled      bool
	savedEvent      domain.Event
	result          domain.Event
	err             error
	receivedEventID domain.EventID
	getResult       domain.Event
	getErr          error
	listResult      []domain.Event
	listErr         error
}

// Save реализует EventRepository и сохраняет аргумент для проверки взаимодействия.
func (repository *fakeEventRepository) Save(event domain.Event) (domain.Event, error) {
	repository.saveCalled = true
	repository.savedEvent = event

	return repository.result, repository.err
}

func (repository *fakeEventRepository) Get(eventID domain.EventID) (domain.Event, error) {
	repository.receivedEventID = eventID

	return repository.getResult, repository.getErr
}

func (repository *fakeEventRepository) List() ([]domain.Event, error) {
	return repository.listResult, repository.listErr
}

// TestEventSaveRepository проверяет успешное сохранение и оба пути ошибок EventService.
func TestEventService_Create(t *testing.T) {
	tests := []struct {
		name          string
		event         domain.Event
		result        domain.Event
		repositoryErr error
		wantErr       error
		wantSave      bool
	}{
		{
			name: "success",
			event: domain.Event{
				Type:    domain.OrderCreated,
				Payload: map[string]any{"status": "ok"},
				Status:  domain.EventPending,
			},
			result: domain.Event{
				ID:      "event-1",
				Type:    domain.OrderCreated,
				Payload: map[string]any{"status": "ok"},
				Status:  domain.EventPending,
			},
			wantErr:  nil,
			wantSave: true,
		},
		{
			name:     "domain error",
			event:    domain.Event{},
			wantErr:  domain.ErrEventTypeEmpty,
			wantSave: false,
		},
		{
			name: "repository error",
			event: domain.Event{
				Type:    domain.OrderCreated,
				Payload: map[string]any{"status": "ok"},
				Status:  domain.EventPending,
			},
			result: domain.Event{
				ID:      "event-1",
				Type:    domain.OrderCreated,
				Payload: map[string]any{"status": "ok"},
				Status:  domain.EventPending,
			},
			repositoryErr: ErrSaveFailed,
			wantErr:       ErrSaveFailed,
			wantSave:      true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// Fake позволяет отдельно проверить факт вызова и ответ repository.
			fake := &fakeEventRepository{
				result: testCase.result,
				err:    testCase.repositoryErr,
			}

			event := NewEventService(fake)
			result, err := event.Create(testCase.event.Type, testCase.event.Payload)

			if fake.saveCalled != testCase.wantSave {
				t.Errorf("saveCalled = %v, want %v", fake.saveCalled, testCase.wantSave)
			}

			if testCase.wantSave {
				assertEventEqual(t, result, testCase.result)
			}

			if testCase.wantErr != nil {
				checkingForValidError(t, err, testCase.wantErr)
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestEventService_Get(t *testing.T) {
	tests := []struct {
		name             string
		repositoryResult domain.Event
		repositoryErr    error
		eventID          domain.EventID
		wantErr          error
	}{
		{
			name: "success",
			repositoryResult: domain.Event{
				ID:      "event-1",
				Type:    domain.OrderCreated,
				Payload: map[string]any{"status": "ok"},
				Status:  domain.EventPending,
			},
			eventID: "event-1",
			wantErr: nil,
		},
		{
			name:             "error",
			repositoryResult: domain.Event{},
			repositoryErr:    domain.ErrEventNotFound,
			eventID:          "event-1",
			wantErr:          domain.ErrEventNotFound,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			fake := &fakeEventRepository{
				getResult: testCase.repositoryResult,
				getErr:    testCase.repositoryErr,
			}

			event := NewEventService(fake)
			result, err := event.Get(testCase.eventID)

			if fake.receivedEventID != testCase.eventID {
				t.Errorf("received event ID = %s, want %s", fake.receivedEventID, testCase.eventID)
			}

			if testCase.wantErr != nil {
				checkingForValidError(t, err, testCase.wantErr)
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assertEventEqual(t, result, testCase.repositoryResult)
		})
	}
}

func TestEventService_List(t *testing.T) {
	tests := []struct {
		name          string
		wantList      []domain.Event
		repositoryErr error
	}{
		{
			name: "success",
			wantList: []domain.Event{
				{
					ID:      "event-1",
					Type:    domain.OrderCreated,
					Payload: map[string]any{"status": "ok"},
					Status:  domain.EventPending,
				},
			},
		},
		{
			name:          "repository error",
			wantList:      nil,
			repositoryErr: ErrRepository,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			fake := &fakeEventRepository{
				listResult: testCase.wantList,
				listErr:    testCase.repositoryErr,
			}

			event := NewEventService(fake)
			result, err := event.List()

			if testCase.repositoryErr != nil {
				checkingForValidError(t, err, testCase.repositoryErr)
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if len(result) != len(testCase.wantList) {
				t.Fatalf("got %d events, want %d", len(result), len(testCase.wantList))
			}

			for index := range result {
				assertEventEqual(t, result[index], testCase.wantList[index])
			}
		})
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

func assertEventEqual(t *testing.T, got, want domain.Event) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("event mismatch:\ngot:  %#v\nwant: %#v", got, want)
	}
}
