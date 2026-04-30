# Circuit Breaker demo на Go (провайдер курса валют + клиент)

Этот каталог изолирует пример с паттерном **Circuit Breaker** от основного проекта rate limiter.

## Что внутри

- `cmd/exchange-provider` — простой HTTP сервис, отдающий курс USD→RUB.
- `cmd/exchange-consumer` — второй сервис, который читает курс из provider через circuit breaker.
- `internal/circuitbreaker` — локальная реализация circuit breaker.

## Быстрый запуск

Из корня репозитория:

```bash
go run ./circuit_breaker/cmd/exchange-provider
```

В другом терминале:

```bash
go run ./circuit_breaker/cmd/exchange-consumer
```

Проверка:

```bash
curl -s "http://localhost:8082/converted?amount=10"
curl -s "http://localhost:8082/breaker"
```

---

## Сценарий воспроизведения состояний

### 1) Closed (нормальный режим)

```bash
curl -s "http://localhost:8082/converted?amount=5"
curl -s "http://localhost:8082/breaker"
```

Ожидается `source=upstream`, `state=closed`.

### 2) Open (апстрим падает)

Перезапустить provider с ошибкой:

```bash
PROVIDER_FORCE_ERROR=true go run ./circuit_breaker/cmd/exchange-provider
```

Затем несколько запросов в consumer:

```bash
for i in {1..5}; do curl -s "http://localhost:8082/converted?amount=3"; echo; done
curl -s "http://localhost:8082/breaker"
```

После достижения `CB_FAILURE_THRESHOLD` breaker станет `open`, и consumer начнет отдавать fallback без вызова апстрима.

### 3) Half-open -> Closed (восстановление)

Вернуть provider в норму:

```bash
PROVIDER_FORCE_ERROR=false go run ./circuit_breaker/cmd/exchange-provider
```

Подождать `CB_OPEN_TIMEOUT_MS` (по умолчанию 3000 мс) и снова вызвать consumer:

```bash
curl -s "http://localhost:8082/converted?amount=7"
curl -s "http://localhost:8082/breaker"
```

При успешной пробе в `half-open` breaker вернется в `closed`.

---

## Конфигурация

### Provider
- `PROVIDER_ADDR` (default `:8081`)
- `USD_RUB` (default `92.15`)
- `PROVIDER_FORCE_ERROR` (default `false`)
- `PROVIDER_DELAY_MS` (default `0`)

### Consumer
- `CONSUMER_ADDR` (default `:8082`)
- `PROVIDER_URL` (default `http://localhost:8081/rate`)
- `UPSTREAM_TIMEOUT_MS` (default `500`)
- `FALLBACK_USD_RUB` (default `90.0`)
- `CB_FAILURE_THRESHOLD` (default `3`)
- `CB_OPEN_TIMEOUT_MS` (default `3000`)
- `CB_HALF_OPEN_MAX_CALLS` (default `1`)

---

## Trade-offs и bottlenecks

- Простая реализация без внешней библиотеки: легко читать, но меньше production-функций.
- Один breaker на весь upstream: проще, но менее гибко.
- Mutex внутри breaker может стать контеншном при очень высоком RPS.
- Фиксированный fallback курс безопасен по доступности, но не всегда актуален по данным.

---

## Тесты

```bash
go test ./...
```

Покрыты:
- переходы состояний circuit breaker,
- success/error сценарии provider,
- поведение consumer в healthy режиме и при открытом breaker.
