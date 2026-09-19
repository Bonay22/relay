package memory

import (
	"fmt"
	"maps"
	"sync"

	"github.com/Bonay22/relay/internal/domain"
)

type EventRepository struct {
	mutex  sync.RWMutex
	events map[domain.EventID]domain.Event
	nextID uint64
	order  []domain.EventID
}

func NewEventRepository() *EventRepository {
	return &EventRepository{
		events: make(map[domain.EventID]domain.Event),
		nextID: 1,
	}
}

func (repository *EventRepository) Save(event domain.Event) (domain.Event, error) {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	generatedID := domain.EventID(fmt.Sprintf("event-%d", repository.nextID))
	repository.nextID++

	event.ID = generatedID

	dbEvent := cloneEvent(event)
	repository.events[generatedID] = dbEvent
	repository.order = append(repository.order, generatedID)

	return cloneEvent(dbEvent), nil
}

func (repository *EventRepository) Get(eventID domain.EventID) (domain.Event, error) {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	event, ok := repository.events[eventID]

	if !ok {
		return domain.Event{}, domain.ErrEventNotFound
	}

	return cloneEvent(event), nil
}

func (repository *EventRepository) List() ([]domain.Event, error) {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	result := make([]domain.Event, 0, len(repository.order))

	for _, id := range repository.order {
		event, ok := repository.events[id]
		if !ok {
			return nil, domain.ErrEventNotFound
		}

		event = cloneEvent(event)
		result = append(result, event)
	}

	return result, nil
}

func cloneEvent(event domain.Event) domain.Event {
	event.Payload = maps.Clone(event.Payload)
	return event
}
