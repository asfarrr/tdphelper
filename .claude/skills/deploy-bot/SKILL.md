---
name: deploy-bot
description: Use when preparing the Go Telegram bot for production — multi-stage Dockerfile, docker-compose with Redis, switching from polling to webhook, GitHub Actions CI, secrets and graceful shutdown.
---

# Деплой бота

## Dockerfile
```dockerfile
FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bot ./cmd/bot

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /bot /bot
ENTRYPOINT ["/bot"]
```

## docker-compose.yml
```yaml
services:
  bot:
    build: .
    env_file: .env
    restart: unless-stopped
    depends_on: [redis]
  redis:
    image: redis:7-alpine
    restart: unless-stopped
```

## Polling vs Webhook
- Polling (`b.Start(ctx)`) — проще, для VPS и одного инстанса.
- Webhook — для нескольких инстансов/serverless:
  ```go
  b, _ := bot.New(token, bot.WithWebhookSecretToken(secret))
  b.SetWebhook(ctx, &bot.SetWebhookParams{URL: publicURL, SecretToken: secret})
  go b.StartWebhook(ctx)
  srv := &http.Server{Addr: ":8080", Handler: b.WebhookHandler(), ReadHeaderTimeout: 5 * time.Second}
  ```
  HTTPS обязателен (reverse proxy / платформа).

## CI (.github/workflows/ci.yml)
```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version-file: go.mod }
      - run: test -z "$(gofmt -l .)"
      - run: go vet ./...
      - run: go test -race ./...
      - uses: golangci/golangci-lint-action@v8
```

## Чек-лист перед релизом
- [ ] Секреты только в env/секретах платформы; `.env` не в git.
- [ ] Graceful shutdown по SIGTERM, логи `slog` в stdout (JSON-хендлер в проде).
- [ ] `WithErrorsHandler`, throttling middleware.
- [ ] Состояния диалогов во внешнем хранилище при >1 инстансе.
- [ ] `restart: unless-stopped` / systemd `Restart=always`.
- [ ] Деплой и push — только после подтверждения пользователя.
