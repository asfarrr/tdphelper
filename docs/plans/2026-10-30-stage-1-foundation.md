# План этапа 1: основа

Цель: данные и расчёты без Telegram-интерфейса. Хранятся материалы, филамент, катушки, журнал движений и настройки; калькулятор выдаёт цену. Основание: [дизайн продукта](2026-10-30-product-design.md).

Каждая задача выполняется по TDD: падающий тест, минимальная реализация, рефакторинг, затем проверка и отдельный коммит:

```bash
gofmt -l . && go vet ./... && go test -race ./... && golangci-lint run ./...
```

## Перед началом

- Создать ветку от актуальной основной ветки. Сейчас каркас лежит в `chore/import-bot-skeleton` и ещё не влит в `master`.
- **Получить подтверждение на зависимость.** Нужен `modernc.org/sqlite` (SQLite без CGO), по классификатору `go get` входит в ASK.

## Задача 1. Очистить линтер

- Добавить комментарии для `main`, `Config`, `Load`, `Start`, `Help`.
- Готово, когда `make lint` проходит.

## Задача 2. Калькулятор

Файлы: `internal/service/pricing.go`, `internal/service/pricing_test.go`.

```go
type QuoteInput struct {
    Lines          []MaterialLine // граммы и цена за кг в копейках
    PrintMinutes   int
    LaborMinutes   int
    MachineKopPerH int64
    LaborKopPerH   int64
    FailureBP      int // запас на брак в базисных пунктах
    MarkupBP       int
    MinKop         int64
    RoundToKop     int64
    DeliveryKop    int64
}

type Quote struct {
    MaterialKop, MachineKop, LaborKop, BufferKop, CostKop, PriceKop, TotalKop int64
}

func CalculateQuote(in QuoteInput) (Quote, error)
```

Табличные тесты:

- Пример из дизайна: себестоимость 666,67 ₽ и цена 900 ₽.
- Несколько материалов.
- Минимальный заказ.
- Округление вверх, в том числе когда цена уже кратна шагу.
- Доставка не входит в наценку.
- Ошибка при отрицательных значениях и браке от 100%.
- Правило округления промежуточных копеек зафиксировано тестом.

## Задача 3. База данных и миграции

Файлы: `internal/storage/db.go`, `internal/storage/migrations/*.sql`, `internal/storage/db_test.go`.

- `Open(ctx, path)` включает WAL и внешние ключи и применяет миграции из `embed.FS`.
- Простая таблица `schema_migrations`, без отдельной библиотеки миграций.
- Тесты на `t.TempDir()`: повторный запуск не меняет схему, внешние ключи работают.

Схема:

| Таблица | Основные поля |
|---|---|
| `materials` | `name`, `machine_kop_per_hour`, `failure_bp`, `hidden` |
| `filaments` | `material_id`, `color_name`, `color_hex`, `brand`, `price_kop_per_kg`, `low_stock_grams`, `hidden` |
| `spools` | `filament_id`, `initial_grams`, `empty_spool_grams`, `remaining_grams`, `status` |
| `stock_movements` | `spool_id`, `delta_grams`, `kind`, `order_id` (пока без внешнего ключа), `note`, `created_at` |
| `settings` | `key`, `value` |

`kind`: `purchase`, `consume`, `waste`, `personal`, `adjust`. Резервы появятся на этапе 5 вместе с заказами.

## Задача 4. Материалы и филамент

Файлы: `internal/service/inventory.go` и тесты, `internal/storage/inventory.go`.

- Интерфейс хранилища объявлен в `service`, реализация в `storage`.
- Создание, изменение, скрытие, список доступных для клиента.
- Тесты: скрытые не попадают в список, некорректный цвет и цена отклоняются, удаления нет, чтобы не ломать историю.

## Задача 5. Катушки и журнал

Там же.

```go
AddSpool(ctx, filamentID, initialGrams, emptySpoolGrams) (Spool, error)
Consume(ctx, spoolID, grams, kind, note) error
AdjustByWeight(ctx, spoolID, weightGrams) (Movement, error)
Available(ctx, filamentID) (int64, error)
```

- Изменение остатка и запись в журнал идут в одной транзакции.
- Списание больше остатка возвращает ошибку.
- Вес меньше пустой катушки возвращает ошибку.
- Инвариант теста: остаток всегда равен сумме движений.
- Параллельные списания не уводят остаток в минус (`-race`).

## Задача 6. Настройки

- Типизированное чтение и запись ставки работы, наценки, минимального заказа и шага округления.
- Значения по умолчанию задаются при первом запуске.
- `service.BuildQuoteInput` собирает вход калькулятора из настроек и материалов; при нескольких материалах берёт самую высокую ставку и самый высокий брак.

## Задача 7. Подключение к приложению

- Добавить `DB_PATH` в `internal/config` и `.env.example`. Значение по умолчанию `data/tdphelper.db`, каталог `data/` в `.gitignore`.
- `cmd/bot/main.go` открывает базу при запуске и корректно закрывает её.
- Добавить инструкцию по бэкапу в `docs/`.
- Обновить `AGENTS.md` (структура) и `docs/HANDOFF.md` (состояние и шаги).

## Критерии завершения

- Все проверки зелёные, `go test -race` чистый.
- Пример из дизайна проходит через настройки, материалы и калькулятор и даёт 900 ₽.
- Бот запускается, создаёт базу и останавливается без ошибок. Запуск с настоящим токеном только с подтверждением владельца.

## Вне этапа

Telegram-интерфейс заказов, Mini App, HTTP API, каталог, резервы.
