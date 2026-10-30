---
name: new-handler
description: Use when adding a new command, text or callback handler to the Go Telegram bot (github.com/go-telegram/bot) — writing the handler, registering it, extending Sender, and covering it with tests.
---

# Новый хендлер

## Шаги
1. **Тест первым** — `internal/handlers/<feature>_test.go` с `fakeSender` (см. `bot-testing`). `go test ./internal/handlers/` — должен упасть.
2. **Хендлер** `internal/handlers/<feature>.go`:
   ```go
   func Stats(ctx context.Context, s Sender, update *models.Update) {
       if update.Message == nil {
           return
       }
       reply(ctx, s, update.Message.Chat.ID, "...")
   }
   ```
3. **Регистрация** в `handlers.Register`:
   ```go
   b.RegisterHandler(bot.HandlerTypeMessageText, "stats", bot.MatchTypeCommand, adapt(b, Stats))
   ```
4. **Новый метод API** (EditMessageText, SendPhoto, ...) → добавь в интерфейс `Sender` и в `fakeSender`.
5. **Зависимости** (сервисы, конфиг) → не глобалы: сделай `type StatsHandler struct{ svc *service.Stats }` с методом нужной сигнатуры, создавай в `Register`.
6. **Логика** длиннее пары строк → `internal/service`, хендлер только вызывает.
7. **Меню команд** (опционально): `b.SetMyCommands(ctx, &bot.SetMyCommandsParams{Commands: []models.BotCommand{...}})` и обновить `/help`.
8. Проверка: `go vet ./... && go test -race ./...`.

## Типы матчинга
| Нужно | Регистрация |
|---|---|
| Команда `/x` | `HandlerTypeMessageText, "x", MatchTypeCommand` |
| Точный текст кнопки | `HandlerTypeMessageText, "Меню", MatchTypeExact` |
| Callback | `HandlerTypeCallbackQueryData, "item:", MatchTypePrefix` |
| Произвольный ввод/медиа | `bot.WithDefaultHandler(...)` или `RegisterHandlerMatchFunc` |
| Только админ | middleware, проверяющая `From.ID` по `cfg.AdminIDs` |

## Ошибки, которых избегать
- Не зарегистрировать хендлер → он молча не срабатывает.
- Разыменовать `update.Message`/`From` без проверки на `nil`.
- Не экранировать пользовательский ввод при HTML.
- Блокирующие операции без `ctx`.
