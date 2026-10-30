// Package handlers contains Telegram update handlers.
// Handlers stay thin: parse the update, call internal/service, reply.
package handlers

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Sender is the subset of *bot.Bot used by handlers; tests substitute a fake.
type Sender interface {
	SendMessage(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error)
}

// Register wires all handlers into the bot.
func Register(b *bot.Bot) {
	b.RegisterHandler(bot.HandlerTypeMessageText, "start", bot.MatchTypeCommand, adapt(b, Start))
	b.RegisterHandler(bot.HandlerTypeMessageText, "help", bot.MatchTypeCommand, adapt(b, Help))
}

type handlerFunc func(ctx context.Context, s Sender, update *models.Update)

func adapt(s Sender, h handlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		h(ctx, s, update)
	}
}
