package handlers

import (
	"context"
	"fmt"
	"html"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Start greets the user and points to the help command.
func Start(ctx context.Context, s Sender, update *models.Update) {
	msg := update.Message
	name := "друг"
	if msg.From != nil {
		name = msg.From.FirstName
	}
	reply(ctx, s, msg.Chat.ID, fmt.Sprintf(
		"Привет, <b>%s</b>! Набери /help, чтобы узнать, что я умею.", html.EscapeString(name)))
}

// Help lists the available commands.
func Help(ctx context.Context, s Sender, update *models.Update) {
	reply(ctx, s, update.Message.Chat.ID, "/start — начать\n/help — помощь")
}

func reply(ctx context.Context, s Sender, chatID int64, text string) {
	_, err := s.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		slog.ErrorContext(ctx, "send message", "chat_id", chatID, "err", err)
	}
}
