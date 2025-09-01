# Rate Limiter Example

This project demonstrates a simple HTTP service with a Redis-backed rate limiter.

## Endpoints
- `GET /v1/users` – list users
- `PATCH /v1/users/{id}/change-password` – change password with limit of two requests per 24h

## Running locally
```
docker-compose up --build
```
This will start Postgres, Redis and the application.

To apply database migrations separately:
```
go run cmd/migrate/main.go
```

## Load test
With the server running, execute:
```
go run cmd/loadtest/main.go
```
The first two requests return `200 OK`, subsequent ones return `429 Too Many Requests`.
