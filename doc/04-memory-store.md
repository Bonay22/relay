# Этап 04. Хранилище в памяти и границы приложения

## Карточка этапа

- **Ветка:** `stage/04-memory-store`
- **Статус:** `in progress`
- **Последний принятый Git checkpoint:** `85f4732`
- **Планируемый коммит подинкремента 5A:** `feat: persist events through HTTP service`
- **Ориентир:** 5 продуктовых инкрементов

## Результат этапа

Relay сохраняет созданные события в памяти, возвращает событие по ID и выдаёт
детерминированный список. Данные живут только до остановки процесса, а HTTP,
application, domain и storage имеют явные границы.

## Учебный scope

### Core

- [экспортированный API пакета](GO-KNOWLEDGE-MAP.md#pkg-06);
- [зависимости пакетов, import cycles и `internal`](GO-KNOWLEDGE-MAP.md#pkg-07);
- [маленький interface и неявная реализация](GO-KNOWLEDGE-MAP.md#iface-01);
- [method sets и удовлетворение interface](GO-KNOWLEDGE-MAP.md#iface-02);
- [consumer-side interface](GO-KNOWLEDGE-MAP.md#iface-07);
- [slice, `append`, aliasing и безопасное копирование](GO-KNOWLEDGE-MAP.md#coll-02);
- [разделяемое внутреннее состояние map/slice](GO-KNOWLEDGE-MAP.md#coll-08);
- [`sync.RWMutex` и critical section](GO-KNOWLEDGE-MAP.md#conc-10);
- [race detector](GO-KNOWLEDGE-MAP.md#test-09);
- [явная передача зависимостей](GO-KNOWLEDGE-MAP.md#arch-03);
- [repository/interface boundary](GO-KNOWLEDGE-MAP.md#arch-04).

### Later

`context.Context` в repository, generics для коллекций, кеширование и
оптимистические блокировки.

Ученик самостоятельно применил `decodeJSON[T any]` в HTTP-тестах. Полное
изучение generics остаётся на этапе 12, включая уже знакомый синтаксис; этот
helper не означает завершения темы.

### Reference

DI-контейнеры, reflection-based ORM, полный DDD/Clean Architecture и lock-free
структуры.

Только Core блокирует завершение этапа. Точные детали следующих инкрементов
объясняются перед их первым применением.

## Закрепляем

Defined types через `EventID`, `errors.Is`, JSON/HTTP, table-driven tests, maps,
pointer semantics и границу transport/domain.

## План инкрементов

| № | Возможность | Новые Core / закрепляем | Проверяемый результат | Статус |
|---:|---|---|---|---|
| 1 | Доменная модель в `internal/domain` | package boundary, `internal`, export, `EventID` | HTTP-поведение не меняется, модель больше не принадлежит `package main` | `done` |
| 2 | Application service и repository contract | consumer-side interface, method sets, dependency injection | service работает через подставляемый repository | `done` |
| 3 | Создание и получение из памяти | map как индекс, `ErrNotFound`, `RWMutex` | событие получает ID и читается по нему | `done` |
| 4 | Детерминированный безопасный список | slices, `append`, порядок, защитные копии | результат стабилен и не раскрывает изменяемое состояние хранилища | `done` |
| 5 | Полный HTTP API и конкурентная проверка | handler/service wiring, GET-маршруты, race detector | POST сохраняет событие, GET возвращает его и список без race | `in progress`: 5A завершён |

## Завершённый подинкремент 5A

- **Статус:** `done`, изменения пока не закоммичены.
- **Результат:** `POST /v1/events` создаёт и сохраняет событие через application
  service, возвращает `201` и назначенный repository ID.
- **Поток:** request DTO → `EventService.Create` → domain constructor → memory
  `Save` → response DTO. `main` собирает repository → service → router.
- **Применённые концепции:** явная композиция зависимостей, handler с полем
  service, method value для `HandleFunc`, интеграционный тест без сети.
- **Acceptance signal:** два независимых запроса возвращают `event-1`/`A-10`
  и `event-2`/`B-20`; `service.Get` подтверждает сохранение каждого события.

## Следующий инкремент

Следующий ограниченный шаг 5B — `GET /v1/events/{id}`: чтение ID из маршрута,
получение события через service, ответы `200` и `404`. Затем остаются список
`GET /v1/events` и отдельная проверка параллельных запросов через race detector.

## Решения и ход реализации

- `internal/domain` выбран как первая явная граница: он может использоваться
  всеми внутренними слоями Relay, но не предназначен как публичная библиотека
  для других модулей.
- ID остаётся нулевым у только что сконструированного события и назначается
  memory repository непосредственно при сохранении.
- Доменная модель и тесты перенесены без зависимости от HTTP; transport явно
  импортирует domain и сохраняет прежний response DTO без поля ID.
- `EventRepository` объявлен со стороны application consumer и содержит реально
  используемые `Save`, `Get` и `List`; `EventService` получает его через
  constructor, а fake с pointer receiver доказывает взаимодействие и пути
  ошибок.
- Domain определяет sentinel `ErrEventNotFound`, а application interface и
  `EventService` предоставляют `Get`, не раскрывая детали memory storage.
- Memory repository создаёт map в constructor, назначает последовательные ID и
  защищает `nextID` вместе с записью через `Lock`; чтение выполняется через
  `RLock` и comma-ok.
- Memory repository рекурсивно копирует JSON-подобный `Payload`: map и slice
  получают новые контейнеры, а скалярные значения копируются как значения.
- В 5A HTTP response DTO получил ID сохранённого события. HTTP handler зависит
  от service, а детали memory repository остаются в composition root.

## Code review

### Обязательные замечания

Исправлено: преждевременное поле `id` удалено из HTTP response DTO. Во втором
инкременте исправлены путь `applecation`, отсутствующая проверка аргумента
`Save` и ошибочный доступ к ожидаемому payload вместо `fake.savedEvent`.
В третьем инкременте исправлены потерянный `error` в сигнатуре `Save`, запись ID
только в ключ map, тест application service в обход `EventService.Get` и
ошибочный memory test, который обращался к элементу пустого slice. Итоговая
реализация соответствует критерию `Save/Get`.

В четвёртом инкременте исправлены вложенная блокировка `RLock`, отброшенные
результаты копирования и тесты, которые сначала проверяли только верхний уровень
map. Финальные тесты отдельно защищают вход `Save` и результаты `Save`, `Get`,
`List`; refactoring pass убрал дублирование map-клонирования и общей fixture.

В 5A исправлены обход service, запись ответа до сохранения, повторное чтение
исчерпанного response body и путаница между ID события и `order_id` в payload.
Refactoring pass разделил проверки HTTP DTO и доменной модели без преобразования
между ними; небольшое повторение helpers сохранено ради ясных контрактов.

### Рекомендации

Нет после refactoring pass.

### Исправления и подтверждение понимания

Ученик добавил проверки zero value `EventID`, разделил test helpers между
пакетами и объяснил независимость response DTO и точную область доступа
`internal`. Для service ученик различает статический `EventRepository` и
динамический `*fakeEventRepository`, понимает pointer receiver и проверяет, что
доменная ошибка останавливает вызов repository.

Для memory repository ученик объясняет назначение map и `make`, причину общей
critical section для счётчика и записи, различает `Lock` и `RLock`, а также
понимает выполнение отложенного unlock перед фактическим возвратом из метода.

Для защитных копий ученик различает отдельный backing array slice и отдельную
вложенную map, применяет рекурсивный type switch и создаёт независимые fixtures
для expected values.

## Проверки

- [x] `gofmt` не оставляет изменений после подинкремента 5A.
- [x] `git diff --check` после подинкремента 5A.
- [x] `go vet ./...` после подинкремента 5A.
- [x] `go test ./...` после подинкремента 5A.
- [x] `go test -race ./...` после подинкремента 5A. Отдельный конкурентный
  HTTP-сценарий ещё предстоит добавить.

## Ретроспектива

- **Что стало понятно:**
- **Что было сложным:**
- **Что вернём в Later:**
- **Осознанно оставленный технический долг:**

## Итог и контрольная точка

Этап начат. Итог будет заполнен после реализации хранилища, HTTP-интеграции и
прохождения race detector.
