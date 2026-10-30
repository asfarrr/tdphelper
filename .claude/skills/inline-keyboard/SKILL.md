---
name: inline-keyboard
description: Use when adding inline or reply keyboards, buttons, pagination or callback query handling to the Go Telegram bot (go-telegram/bot models.InlineKeyboardMarkup, callback data encoding).
---

# Клавиатуры и callback'и

## Callback data — типизированно, не сырые строки
```go
// internal/keyboards/item.go
const itemPrefix = "item:"

type ItemCallback struct {
    Action string // "view" | "delete"
    ID     int64
}

func (c ItemCallback) Encode() string { return fmt.Sprintf("%s%s:%d", itemPrefix, c.Action, c.ID) }

func ParseItemCallback(data string) (ItemCallback, error) { /* strings.Cut + strconv, ошибка на мусор */ }

func Items(items []Item) models.InlineKeyboardMarkup {
    rows := [][]models.InlineKeyboardButton{}
    for _, it := range items {
        rows = append(rows, []models.InlineKeyboardButton{{
            Text: it.Title, CallbackData: ItemCallback{"view", it.ID}.Encode(),
        }})
    }
    return models.InlineKeyboardMarkup{InlineKeyboard: rows}
}
```
Отправка: `SendMessageParams{..., ReplyMarkup: keyboards.Items(items)}`.

## Хендлер
```go
b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "item:", bot.MatchTypePrefix, adapt(b, OnItem))

func OnItem(ctx context.Context, s Sender, u *models.Update) {
    cq := u.CallbackQuery
    // всегда отвечаем, иначе «часики» на кнопке
    defer s.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: cq.ID})
    cb, err := keyboards.ParseItemCallback(cq.Data)
    if err != nil { return }
    if cq.Message.Message == nil { return } // сообщение недоступно (старое)
    s.EditMessageText(ctx, &bot.EditMessageTextParams{
        ChatID: cq.Message.Message.Chat.ID, MessageID: cq.Message.Message.ID, Text: "...",
    })
}
```
(`AnswerCallbackQuery`/`EditMessageText` добавь в интерфейс `Sender`.)

## Правила
- Callback data ≤ 64 байта: короткий prefix + id, не данные целиком.
- Не доверяй callback data: проверь, что объект принадлежит `cq.From.ID`.
- Редактирование тем же текстом даёт ошибку «message is not modified» — обрабатывай.
- Пагинация: `item:page:<n>` + кнопки ◀️ ▶️; не больше 100 кнопок.
- Reply-клавиатура: `models.ReplyKeyboardMarkup{ResizeKeyboard: true}`, убрать — `models.ReplyKeyboardRemove{RemoveKeyboard: true}`.

## Тест
Encode/Parse — round-trip table-driven тест + мусорные строки; клавиатура — проверка количества кнопок и их data.
