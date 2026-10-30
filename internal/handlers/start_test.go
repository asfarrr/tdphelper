package handlers

import (
	"context"
	"strings"
	"testing"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type fakeSender struct {
	sent []*bot.SendMessageParams
}

func (f *fakeSender) SendMessage(_ context.Context, p *bot.SendMessageParams) (*models.Message, error) {
	f.sent = append(f.sent, p)
	return &models.Message{}, nil
}

func textUpdate(text string, from *models.User) *models.Update {
	return &models.Update{Message: &models.Message{
		Text: text,
		Chat: models.Chat{ID: 42},
		From: from,
	}}
}

func TestStartGreetsUserAndEscapesName(t *testing.T) {
	s := &fakeSender{}

	Start(context.Background(), s, textUpdate("/start", &models.User{FirstName: "<Влад>"}))

	if len(s.sent) != 1 {
		t.Fatalf("sent %d messages, want 1", len(s.sent))
	}
	if got := s.sent[0].Text; !strings.Contains(got, "&lt;Влад&gt;") {
		t.Errorf("text %q does not contain escaped name", got)
	}
	if s.sent[0].ChatID != int64(42) {
		t.Errorf("ChatID = %v, want 42", s.sent[0].ChatID)
	}
}

func TestStartWithoutUser(t *testing.T) {
	s := &fakeSender{}

	Start(context.Background(), s, textUpdate("/start", nil))

	if !strings.Contains(s.sent[0].Text, "друг") {
		t.Errorf("text %q, want fallback name", s.sent[0].Text)
	}
}

func TestHelpListsCommands(t *testing.T) {
	s := &fakeSender{}

	Help(context.Background(), s, textUpdate("/help", nil))

	for _, cmd := range []string{"/start", "/help"} {
		if !strings.Contains(s.sent[0].Text, cmd) {
			t.Errorf("help text missing %s", cmd)
		}
	}
}
