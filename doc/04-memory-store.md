# Этап 04. Хранилище в памяти и границы приложения

## Карточка этапа

- **Ветка:** `stage/04-memory-store`
- **Статус:** `in progress`
- **Предыдущий checkpoint:** `aa2cfad`
- **Планируемый коммит:** `feat: add in-memory event storage`
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
| 4 | Детерминированный безопасный список | slices, `append`, порядок, защитные копии | результат стабилен и не раскрывает изменяемое состояние хранилища | `in progress` |
| 5 | Полный HTTP API и конкурентная проверка | handler/service wiring, GET-маршруты, race detector | POST сохраняет событие, GET возвращает его и список без race | `planned` |

## Текущий инкремент

- **Статус:** `in progress`.
- **Добавляемая возможность:** repository возвращает события списком в порядке
  сохранения и не раскрывает вызывающему коду верхний уровень сохранённого
  `Payload`.
- **Зачем она нужна:** порядок обхода map не задаёт порядок создания, а
  возвращаемая map без копии позволяет обойти методы repository и изменить его
  внутреннее состояние.
- **Новые концепции:** slice как упорядоченная последовательность, `append`,
  копирование структурного значения и `maps.Clone`.
- **Что закрепляем:** `RLock`/`Lock`, application delegation, table-driven tests
  и map aliasing.
- **Что намеренно отложено:** рекурсивная копия вложенных JSON map/slice до
  второй части инкремента; HTTP wiring и конкурентный тест — до инкремента 5.
- **Критерий первой части:** `List` возвращает `event-1`, затем `event-2`; пустой
  repository возвращает пустой список; изменение исходного или возвращённого
  верхнего уровня `Payload` не меняет сохранённое событие.

## Необходимые объяснения и примеры

Map остаётся индексом для `Get`, а отдельный `[]domain.EventID` сохраняет порядок
ID. `Save` добавляет новый ID через `append` под тем же `Lock`; `List` под
`RLock` проходит по slice и читает соответствующие события из map.

Поток первой части четвёртого инкремента:

```text
memory.Save → записать Event в map → append ID в order
memory.List → пройти order → прочитать events[ID] → новый []Event
```

Присваивание `Event` копирует структуру, но не содержимое `Payload`. На первой
части helper копирует верхний уровень payload через `maps.Clone` при входе в
repository и при возврате из `Get`/`List`. Вложенные map и slice остаются общей
ссылочной частью до следующего задания этого же инкремента.

## Задание

1. Расширить application interface методом `List() ([]domain.Event, error)` и
   добавить делегирующий `EventService.List`; fake должен проверить результат и
   ошибку делегирования.
2. Добавить в memory repository поле `order []domain.EventID`; `Save` добавляет
   новый ID через `append` внутри существующей write critical section.
3. Реализовать `List() ([]domain.Event, error)`: под `RLock` пройти ID в `order`
   и собрать новый slice событий в том же порядке.
4. Добавить небольшой helper копирования `Event`, который заменяет `Payload` на
   `maps.Clone(event.Payload)`. Сохранять копию события и возвращать копии из
   `Get` и `List`.
5. Тестами доказать порядок, пустой список и отсутствие top-level aliasing со
   входным payload и результатами `Get`/`List`.
6. Пока не менять HTTP и не реализовывать рекурсивное копирование вложенных
   JSON-коллекций.

Acceptance signal: application получает стабильный список, а изменение
top-level payload у исходного или возвращённого события не меняет состояние
memory repository.

## Решения и ход реализации

- `internal/domain` выбран как первая явная граница: он может использоваться
  всеми внутренними слоями Relay, но не предназначен как публичная библиотека
  для других модулей.
- ID остаётся нулевым у только что сконструированного события и назначается
  memory repository непосредственно при сохранении.
- Доменная модель и тесты перенесены без зависимости от HTTP; transport явно
  импортирует domain и сохраняет прежний response DTO без поля ID.
- `EventRepository` объявлен со стороны application consumer и содержит реально
  используемые `Save` и `Get`; `EventService` получает его через constructor, а
  fake с pointer receiver доказывает взаимодействие и пути ошибок.
- Domain определяет sentinel `ErrEventNotFound`, а application interface и
  `EventService` предоставляют `Get`, не раскрывая детали memory storage.
- Memory repository создаёт map в constructor, назначает последовательные ID и
  защищает `nextID` вместе с записью через `Lock`; чтение выполняется через
  `RLock` и comma-ok.
- Копирование вложенного `Payload` намеренно оставлено следующему инкременту:
  сейчас возвращаемое структурное значение всё ещё разделяет внутреннюю map.

## Code review

### Обязательные замечания

Исправлено: преждевременное поле `id` удалено из HTTP response DTO. Во втором
инкременте исправлены путь `applecation`, отсутствующая проверка аргумента
`Save` и ошибочный доступ к ожидаемому payload вместо `fake.savedEvent`.
В третьем инкременте исправлены потерянный `error` в сигнатуре `Save`, запись ID
только в ключ map, тест application service в обход `EventService.Get` и
ошибочный memory test, который обращался к элементу пустого slice. Итоговая
реализация соответствует критерию `Save/Get`.

### Рекомендации

Нет.

### Исправления и подтверждение понимания

Ученик добавил проверки zero value `EventID`, разделил test helpers между
пакетами и объяснил независимость response DTO и точную область доступа
`internal`. Для service ученик различает статический `EventRepository` и
динамический `*fakeEventRepository`, понимает pointer receiver и проверяет, что
доменная ошибка останавливает вызов repository.

Для memory repository ученик объясняет назначение map и `make`, причину общей
critical section для счётчика и записи, различает `Lock` и `RLock`, а также
понимает выполнение отложенного unlock перед фактическим возвратом из метода.

## Проверки

- [x] `gofmt` не оставляет изменений после инкремента 3.
- [x] `go vet ./...` после инкремента 3.
- [x] `go test ./...` после инкремента 3.
- [x] `go test -race ./...` после инкремента 3.

## Ретроспектива

- **Что стало понятно:**
- **Что было сложным:**
- **Что вернём в Later:**
- **Осознанно оставленный технический долг:**

## Итог и контрольная точка

Этап начат. Итог будет заполнен после реализации хранилища, HTTP-интеграции и
прохождения race detector.
