---
name: bot-testing
description: Use when writing or fixing tests for the Go Telegram bot — fake Sender for handlers, table-driven tests, conversation state flows, keyboards and callback data, race detection.
---

# Тестирование бота

Запуск: `go test -race ./...`. Тесты лежат рядом с кодом (`*_test.go`, тот же пакет).

## Фейковый Sender
Хендлеры зависят от интерфейса `Sender`, а не от `*bot.Bot` — поэтому в тестах сеть не нужна.
```go
type fakeSender struct {
    sent    []*bot.SendMessageParams
    answers []*bot.AnswerCallbackQueryParams
    err     error // чтобы проверить обработку ошибок API
}

func (f *fakeSender) SendMessage(_ context.Context, p *bot.SendMessageParams) (*models.Message, error) {
    f.sent = append(f.sent, p)
    return &models.Message{}, f.err
}
```
При добавлении метода в `Sender` — добавь его и в фейк (общий `fakeSender` живёт в `start_test.go`).

## Table-driven хендлер
```go
func TestEcho(t *testing.T) {
    tests := []struct {
        name, in, want string
    }{
        {"plain", "hi", "hi"},
        {"escapes html", "<b>", "&lt;b&gt;"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            s := &fakeSender{}
            Echo(t.Context(), s, textUpdate(tt.in, &models.User{ID: 1}))
            if got := s.sent[0].Text; got != tt.want {
                t.Errorf("got %q, want %q", got, tt.want)
            }
        })
    }
}
```

## Callback-апдейт
```go
u := &models.Update{CallbackQuery: &models.CallbackQuery{
    ID: "1", Data: "item:view:5", From: models.User{ID: 1},
    Message: models.MaybeInaccessibleMessage{Message: &models.Message{ID: 10, Chat: models.Chat{ID: 1}}},
}}
```
Проверь, что `answers` не пустой и что мусорный `Data` не роняет хендлер.

## Диалоги
Используй реальный `state.MemoryStore`, прогоняй сценарий апдейт за апдейтом и проверяй шаг + данные сессии. Параллельных пользователей — через `t.Parallel()`/горутины под `-race`.

## Принципы
- Проверяй поведение: текст, chat id, клавиатуру, состояние, вызовы сервисов.
- Сервисы — без Telegram и без фейков, где возможно.
- Каждый баг → сначала воспроизводящий тест.
- Никаких реальных запросов к Telegram API.
