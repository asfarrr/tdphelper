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

## Классификатор рисков (обязателен в любом режиме, включая авторежим)

Перед каждой командой определи класс. Сомневаешься — бери более строгий. Если несколько команд в цепочке, класс определяет самая опасная.

### 🔴 BLOCK — никогда, даже если кажется нужным
Не выполняй и не ищи обходных путей. Объясни, зачем это нужно, и попроси пользователя сделать это самому.
- Force push в любой форме: `--force`, `--force-with-lease`, `-f`, `+ref`, `--mirror`.
- Переписывание истории: `filter-branch`, `filter-repo`, `reflog expire`, `update-ref -d`.
- Пропуск проверок: `--no-verify`, смена `core.hooksPath`, правка глобального git config.
- `rm -r` по `/`, `~`, `$HOME`, `.`, `..`, `*`, `.git`; `sudo`, `mkfs`, `dd` на устройства, `chmod -R 777`.
- Секреты: чтение, вывод, копирование `.env*` (кроме `.env.example`), `BOT_TOKEN`, дамп окружения (`env`, `printenv`), токены и секреты GitHub. Секреты не попадают в код, логи, коммиты и ответы.
- Прямые вызовы `api.telegram.org`: это боевой токен и реальные пользователи.
- Выполнение скачанного кода: `curl ... | sh`.
- Удаление репозитория (`gh repo delete`).
- Ослабление защиты: правка `.claude/settings.json`, `.claude/hooks/`, `disableAllHooks`.

### 🟡 ASK — только после явного «да» в этом диалоге
Покажи точную команду и что будет потеряно или изменено. Разрешение действует на одну операцию и не переносится на следующие.
- `rm -rf`, `rm -f`, `find -delete`: сначала посмотри, что удаляется.
- Потеря локальных изменений: `git reset --hard`, `git clean -f`, `git branch -D`, `git stash drop/clear`, `git checkout -- .`, `git restore .`.
- `git push` (любой), `git rebase`, `gh pr merge`, `gh release`, `gh workflow run`.
- Деплой и инфраструктура: `docker push`, `docker compose down -v`, `docker system prune`, `kubectl`, `terraform`, `helm`.
- Данные: миграции и `DROP`/`TRUNCATE`/`DELETE` без `WHERE` на любой базе, кроме локальной тестовой.
- Запуск бота на настоящем токене, рассылки, отправка сообщений реальным пользователям.
- Новые или обновлённые зависимости (`go get`), изменение CI.

### 🟢 ALLOW — без подтверждения
Чтение кода, `go test`, `go vet`, `go build`, `gofmt`, `golangci-lint`, `git status/diff/log`, правки кода в рабочей ветке, локальные коммиты, когда их попросили.

### Если действие заблокировано
Блок — это решение, а не препятствие. Не обходи его другой командой, скриптом, `bash -c`, кодировкой или через субагента: сообщи пользователю. Субагенты подчиняются этим же правилам.

Правила частично применяются автоматически: хук `.claude/hooks/guard-bash.sh` и `permissions.deny` в `.claude/settings.json`. Хук ловит не всё, поэтому отсутствие блокировки не значит, что действие разрешено. Этот список главнее.

## Проектные агенты и скиллы
- Агенты (`.claude/agents/`): bot-architect, go-bot-developer, bot-tester, bot-security-reviewer, deploy-engineer.
- Скиллы (`.claude/skills/`): new-handler, conversation-state, inline-keyboard, bot-testing, deploy-bot.
