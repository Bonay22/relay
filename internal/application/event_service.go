package application

import "github.com/Bonay22/relay/internal/domain"

// EventRepository задаёт минимальный контракт хранения, необходимый EventService.
// Интерфейс объявлен на стороне потребителя, поэтому storage не зависит от application.
type EventRepository interface {
	Save(domain.Event) (domain.Event, error)
	Get(eventID domain.EventID) (domain.Event, error)
	List() ([]domain.Event, error)
}

// EventService координирует доменное создание события и его сохранение.
type EventService struct {
	repository EventRepository
}

// NewEventService явно получает repository, чтобы зависимость можно было подменить в тесте.
func NewEventService(repository EventRepository) *EventService {
	return &EventService{
		repository: repository,
	}
}

// Create проверяет доменные правила до обращения к repository.
// Результат и ошибка Save возвращаются без изменения.
func (service *EventService) Create(eventType domain.EventType, payload map[string]any) (domain.Event, error) {
	event, err := domain.NewEvent(eventType, payload)
	if err != nil {
		return domain.Event{}, err
	}

	event, err = service.repository.Save(event)

	return event, err
}

func (service *EventService) Get(eventID domain.EventID) (domain.Event, error) {
	return service.repository.Get(eventID)
}

func (service *EventService) List() ([]domain.Event, error) {
	return service.repository.List()
}
