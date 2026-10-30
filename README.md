# tdphelper

Telegram-бот на Go и [go-telegram/bot](https://github.com/go-telegram/bot).

## Быстрый старт

```bash
cp .env.example .env   # вписать BOT_TOKEN от @BotFather
make run
```

Нужны Go 1.27+, для `make lint` ещё `golangci-lint`.

## Для разработчиков и ИИ-агентов

- [AGENTS.md](AGENTS.md): правила проекта и классификатор рисков. Обязательны для любого инструмента.
- [docs/HANDOFF.md](docs/HANDOFF.md): текущее состояние, решения и следующие шаги.
