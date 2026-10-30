# Telegram Bot — инструкции для Claude

## Стек
- Go 1.27, библиотека **github.com/go-telegram/bot** (handlers на `context`, без внешних зависимостей)
- Конфиг — env-переменные (`internal/config`), логи — `log/slog`
- Тесты — стандартный `testing` (table-driven, фейки вместо моков), линт — `golangci-lint` (`.golangci.yml`)

## Структура
```
cmd/bot/main.go       # точка входа: конфиг, логгер, bot.New, handlers.Register, b.Start
internal/
  config/             # Load() из env: BOT_TOKEN, LOG_LEVEL, ADMIN_IDS
  handlers/           # хендлеры; Register() подключает всё к боту
  keyboards/          # фабрики клавиатур и кодирование callback data
  service/            # бизнес-логика, не зависит от Telegram API
  state/              # хранилище состояний многошаговых диалогов
docs/plans/           # дизайн-доки и планы (superpowers: brainstorming / writing-plans)
```

## Команды
```bash
cp .env.example .env    # вписать BOT_TOKEN
make run                # go run ./cmd/bot c переменными из .env
make test               # go test -race ./...
make lint               # golangci-lint run ./...
make check              # fmt + test + lint
```

## Рабочий процесс (superpowers)
Плагин superpowers включён в `.claude/settings.json`. Следуй его скиллам:
1. Новая фича → `superpowers:brainstorming` → дизайн в `docs/plans/`.
2. `superpowers:writing-plans` → пошаговый план.
3. Реализация через `superpowers:test-driven-development` (сначала падающий тест).
4. Баги → `superpowers:systematic-debugging`.
5. Перед завершением → `superpowers:verification-before-completion` (`make test`, `go vet`, lint — зелёные).

## Правила кода
- Хендлеры имеют сигнатуру `func(ctx, s Sender, update *models.Update)` и подключаются через `adapt()` в `handlers.Register`. `Sender` — узкий интерфейс над `*bot.Bot`; расширяй его нужными методами, тесты используют фейк.
- Хендлеры тонкие: разбор апдейта → вызов `internal/service` → ответ. Логика — в сервисах.
- Всегда проверяй `nil` у `update.Message`, `update.CallbackQuery`, `From`.
- ParseMode HTML: пользовательский ввод — через `html.EscapeString`.
- Ошибки отправки логируй через `slog.ErrorContext`, не паникуй; `ctx` пробрасывай везде.
- Callback data ≤ 64 байт, формат `prefix:action:id`; в callback-хендлерах всегда `AnswerCallbackQuery`.
- Никогда не логируй и не коммить `BOT_TOKEN`; `.env` не читать.

## Проектные агенты и скиллы
- Агенты (`.claude/agents/`): bot-architect, go-bot-developer, bot-tester, bot-security-reviewer, deploy-engineer.
- Скиллы (`.claude/skills/`): new-handler, conversation-state, inline-keyboard, bot-testing, deploy-bot.
