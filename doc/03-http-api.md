# Этап 03. Первый HTTP API

## Карточка этапа

- **Ветка:** `stage/03-http-api`
- **Статус:** `done`
- **Предыдущий checkpoint:** `2658f0a`
- **Планируемый коммит:** `feat: add first Relay HTTP API`
- **Ориентир:** 5 продуктовых инкрементов

## Результат этапа

Relay запускает HTTP-сервер, отвечает на `GET /health` и принимает корректный
JSON через `POST /v1/events`. Ошибочный запрос получает предсказуемый JSON-ответ
с кодом `400`, а HTTP-контракт защищён тестами без настоящего сетевого порта.

## Учебный scope

### Core

- [HTTP request/response, методы, заголовки и status codes](GO-KNOWLEDGE-MAP.md#http-01);
- [`net/http`, handler и `ServeMux`](GO-KNOWLEDGE-MAP.md#http-02);
- [тело запроса и владение им](GO-KNOWLEDGE-MAP.md#http-03);
- [успешный JSON-ответ и error envelope](GO-KNOWLEDGE-MAP.md#http-04);
- [`encoding/json`, Encoder и Decoder](GO-KNOWLEDGE-MAP.md#enc-01);
- [JSON struct tags, неизвестные поля и числа](GO-KNOWLEDGE-MAP.md#enc-02);
- [struct tags как метаданные](GO-KNOWLEDGE-MAP.md#struct-03);
- [обычный `switch`](GO-KNOWLEDGE-MAP.md#ctrl-04);
- [`httptest.ResponseRecorder`](GO-KNOWLEDGE-MAP.md#http-09);
- [граница transport/domain](GO-KNOWLEDGE-MAP.md#arch-02).

### Later

Middleware, аутентификация, idempotency, потоковый JSON и настраиваемые
marshalers.

### Reference

Сторонние routers, WebSocket, HTTP/2 и низкоуровневые интерфейсы наподобие
`http.Hijacker`.

Только Core блокирует завершение этапа. Темы вводятся по текущему продуктовому
инкременту, а не изучаются заранее одним большим блоком.

## Закрепляем

Конструктор и доменные ошибки, структуры, maps/`any`, ранние возвраты,
table-driven tests и разделение ответственности.

## План инкрементов

| № | Возможность | Новые Core / закрепляем | Проверяемый результат | Статус |
|---:|---|---|---|---|
| 1 | Тестируемый router и `GET /health` | request/response, handler, `ServeMux`, JSON, `httptest` | тест получает `200`, JSON content type и `{"status":"ok"}` | `done` |
| 2 | Безопасное чтение JSON-запроса | DTO, struct tags, Decoder, unknown fields, Unicode и числа | корректное тело читается, некорректное отклоняется | `done` |
| 3 | Создание события через HTTP | transport/domain boundary, `NewEvent`, `201 Created` | `POST /v1/events` возвращает созданное событие | `done` |
| 4 | Единый ошибочный ответ | `switch`, `errors.Is`, error envelope, `400 Bad Request` | ожидаемые ошибки имеют стабильный JSON-контракт | `done` |
| 5 | Запуск сервера и полный handler-набор | `ListenAndServe`, wiring, закрепление `httptest` | `main` запускает API, все HTTP-тесты проходят | `done` |

## Завершённый инкремент: тестируемый router и `GET /health`

`newRouter` создаёт отдельный `ServeMux` и связывает `GET /health` с
`healthHandler`. Handler формирует ответ в порядке headers → status → body и
кодирует `{"status":"ok"}` через `json.Encoder`.

`TestHealth` создаёт request и recorder, вызывает `router.ServeHTTP`, после чего
ServeMux выбирает маршрут и вызывает handler. Тест защищает status `200`, JSON
content type и body с переводом строки от `Encode`.

## Завершённый шаг 2A: DTO и строгий decoder

`createEventRequest` описывает только разрешённые поля внешнего JSON и не
подменяет доменный `Event`. `decodeCreateEventRequest` принимает минимальный
`io.Reader`, включает `DisallowUnknownFields` и заполняет DTO через указатель.

Табличный тест защищает корректный объект, malformed JSON и неизвестное
верхнеуровневое поле. Missing `type` не считается ошибкой decoder: поле получает
zero value и будет проверено доменной логикой позже.

## Завершённый инкремент: безопасное чтение JSON

Decoder принимает только один JSON-объект и отклоняет malformed JSON,
неизвестные поля и второе JSON-значение. `UseNumber` сохраняет точное текстовое
представление чисел в `map[string]any`, Unicode-строки проходят без изменений.
Повторный `Decode` выполняется тем же decoder и должен вернуть `io.EOF`.

Табличный тест проверяет все пути, sentinel error для второго объекта,
`json.Number` через type assertion comma-ok и Unicode-значение.

## Завершённый инкремент: создание события через HTTP

`newRouter` собирает оба маршрута Relay, а тесты вызывают ту же полную
конфигурацию. `POST /v1/events` декодирует request DTO, явно преобразует строку
в `EventType`, вызывает `NewEvent` и кодирует отдельный response DTO.

Успешный handler-тест защищает `201 Created`, JSON content type, тип и начальный
статус события, payload и ненулевое время создания. Преобразование в
`EventType` не валидирует строку; текущее доменное правило отклоняет только
пустой тип.

## Завершённый инкремент: единый ошибочный ответ

`writeErrorResponse` формирует единый JSON envelope в порядке headers → status →
body. Decoder errors получают публичное сообщение `invalid request body`, а
доменная ошибка классифицируется expressionless `switch` через `errors.Is`.
Известный пустой тип становится `400`; ветка `default` возвращает безопасный
`500 internal server error` и не продолжает успешный путь.

Табличный handler-тест защищает malformed JSON и пустой `type`, проверяя status,
JSON content type и декодированный envelope. Успешный POST и health check не
изменились.

## Завершённый инкремент: запуск HTTP-сервера

`main` больше не создаёт демонстрационные события. Он собирает полный router,
печатает startup-сообщение и передаёт `http.Handler` блокирующему
`http.ListenAndServe` на `127.0.0.1:8080`. Ошибка запуска или остановки
завершает процесс через `log.Fatalf`.

Реальный локальный сервер проверен запросами к health, успешному POST,
malformed POST и POST с пустым типом. Ответы совпали с handler-тестами.

## Необходимые объяснения и примеры

`http.ListenAndServe(addr, handler)` открывает сетевой адрес, блокирует `main` и
передаёт каждый HTTP-request указанному handler. `newRouter()` возвращает
`http.Handler`, поэтому его результат напрямую подходит серверу. Возвращённая
ошибка означает, что сервер не смог запуститься или прекратил работу; `main`
должен завершить процесс с ненулевым статусом.

## Итоговая проверка

Acceptance signal выполнен: реальные локальные HTTP-запросы получили `200`,
`201` и стабильные `400` envelopes; форматирование, vet, обычные тесты и race
detector прошли.

## Решения и ход реализации

- Router создаётся отдельной функцией: это позволяет тестировать тот же HTTP
  контракт, который позже будет передан серверу.
- Первый инкремент не смешивает health check с приёмом событий.
- Transport DTO отделяется от доменного `Event`: JSON-контракт и доменные
  правила имеют разные причины для изменения.
- Decoder helper принимает `io.Reader`, поэтому одинаково работает с `r.Body` и
  `strings.Reader` из теста.
- Один decoder выполняет оба `Decode`, потому что часть входа может находиться в
  его внутреннем буфере; только `io.EOF` подтверждает отсутствие второго JSON.
- `UseNumber` сохраняет числа внутри `any` как `json.Number`, не выполняя
  автоматического выбора между Go-типами `int` и `int64`.
- `newRouter` без параметров владеет полной таблицей маршрутов; тесты не
  подменяют production wiring своей конфигурацией.
- Request DTO, доменный `Event` и response DTO имеют разные причины изменения;
  HTTP-ответ является копией данных и не позволяет клиенту изменить объект в
  памяти Relay.
- Error envelope не раскрывает сырые decoder/internal errors; общий `return`
  после `switch` завершает все доменные error-пути.
- Для первого запуска используется минимальный `http.ListenAndServe`;
  `http.Server` с timeouts остаётся в запланированном этапе 05.

## Code review

### Обязательные замечания

Нет. Все пять инкрементов приняты.

### Рекомендации

Нет.

### Исправления и подтверждение понимания

Опечатка `healthHeandler` исправлена на `healthHandler`. Ученик объяснил, почему
`http.ResponseWriter` удовлетворяет `io.Writer`, порядок фиксации status code и
цепочку вызовов: тест вызывает `ServeHTTP`, ServeMux выбирает маршрут и вызывает
handler. В шаге 2A добавлены явная проверка неожиданной ошибки и assertion для
`Type`; ученик различает malformed, unknown и missing поля. В шаге 2B ожидание
sentinel error вынесено из имени subtest в отдельное поле таблицы. Ученик
объяснил внутренний буфер decoder, смысл `io.EOF` и различие между type
assertion и преобразованием типа.
В третьем инкременте исправлены пустое тестовое тело, декодирование ответа через
request helper и попытка сравнить map. Router возвращён к роли сборщика всей
таблицы маршрутов. Ученик объяснил, почему подставляемые тестами маршруты не
защищали production wiring, а также различил преобразование типа, доменную
валидацию и текущий malformed-response.
В четвёртом инкременте исправлены продолжение handler после `default` и тест,
который называл unknown field malformed JSON. Ученик различает выход из
`switch` и функции, синтаксис JSON и соответствие DTO, а также публичную и
внутреннюю ошибку.
В пятом инкременте демонстрационный код заменён на composition root с реальным
HTTP-сервером. Исправлена опечатка в `address`; ученик объяснил блокирующий
вызов, роли теста и `net/http` при вызове `ServeHTTP`, а также host и port.

## Проверки

- [x] `gofmt` не оставляет изменений.
- [x] `go vet ./...`
- [x] `go test ./...`
- [x] `go test -race ./...`
- [x] Реальные запросы к `127.0.0.1:8080`: health, успешный POST, malformed POST
  и пустой `type`.

## Ретроспектива

- **Что стало понятно:** router, его таблица маршрутов и цепочка вызова handler.
- **Что было сложным:** `json.NewDecoder` и `json.NewEncoder`.
- **Что вернём в Later:** повторение Encoder/Decoder; middleware,
  authentication, idempotency и расширенная настройка JSON.
- **Осознанно оставленный технический долг:** нет ограничения размера request
  body; произвольный непустой `EventType` пока допустим; ошибки записи JSON не
  логируются; `ListenAndServe` ещё не имеет timeouts и graceful shutdown.

## Итог и контрольная точка

Этап завершён и готов к коммиту `feat: add first Relay HTTP API`. Relay запускает
локальный HTTP-сервер, обслуживает health и создание событий, строго читает JSON
и возвращает стабильные успешные и ошибочные ответы. Коммит создаётся только
после отдельного явного подтверждения ученика.
