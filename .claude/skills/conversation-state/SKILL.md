---
name: conversation-state
description: Use when building a multi-step conversation in the Go Telegram bot (forms, wizards, questionnaires) — per-user state storage, step transitions, /cancel, routing free-text input by state.
---

# Многошаговый диалог

go-telegram/bot не имеет встроенного FSM — состояние храним сами в `internal/state`.

## Хранилище
```go
// internal/state/store.go
type Step string

type Session struct {
    Step Step
    Data map[string]string
}

type Store interface {
    Get(ctx context.Context, userID int64) (Session, bool, error)
    Set(ctx context.Context, userID int64, s Session) error
    Clear(ctx context.Context, userID int64) error
}

// MemoryStore — для dev и тестов; в проде Redis-реализация того же интерфейса.
type MemoryStore struct {
    mu sync.Mutex
    m  map[int64]Session
}
```
Хендлеры выполняются конкурентно — `MemoryStore` обязательно под mutex.

## Роутинг ввода
- Команда старта (`/order`) → `Set(userID, Session{Step: StepName})` + вопрос.
- `/cancel` зарегистрирован как отдельная команда → `Clear` из любого шага.
- Свободный текст → default handler: `Get` сессию, `switch sess.Step` → обработчик шага; нет сессии → обычный fallback.
```go
type OrderFlow struct {
    store state.Store
    svc   *service.Orders
}

func (f *OrderFlow) OnText(ctx context.Context, s Sender, u *models.Update) {
    sess, ok, err := f.store.Get(ctx, u.Message.From.ID)
    if err != nil || !ok { /* fallback */ return }
    switch sess.Step {
    case StepName:  // валидация, Data["name"]=..., Step=StepPhone
    case StepPhone: // ...
    }
}
```

## Чек-лист
- [ ] `/cancel` работает из любого шага.
- [ ] На каждом шаге есть ответ на невалидный ввод (не текст, пустой, слишком длинный).
- [ ] Валидация в `internal/service`, покрыта table-driven тестами.
- [ ] В конце `Clear`.
- [ ] TTL/очистка брошенных сессий.
- [ ] Тест проходит весь сценарий на `MemoryStore` и проверяет итоговые данные; `go test -race` чистый.
